/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Free Trial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Free-Trial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package blob

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	aws2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	_ "gocloud.dev/blob/s3blob"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"kubestash.dev/apimachinery/apis"
	storageapi "kubestash.dev/apimachinery/apis/storage/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	gcsPrefix                    = "gs://"
	azurePrefix                  = "azblob://"
	localPrefix                  = "file:///"
	credentialsDir               = apis.TempDirMountPath + "/credentials"
	azureStorageAccount          = "AZURE_STORAGE_ACCOUNT"
	azureStorageKey              = "AZURE_STORAGE_KEY"
	googleServiceAccountJsonKey  = "GOOGLE_SERVICE_ACCOUNT_JSON_KEY"
	googleApplicationCredentials = "GOOGLE_APPLICATION_CREDENTIALS"
	azureAccountKey              = "AZURE_ACCOUNT_KEY"
	caCertData                   = "CA_CERT_DATA"
	awsAccessKeyId               = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKey           = "AWS_SECRET_ACCESS_KEY"
)

type Blob struct {
	prefix        string
	storageURL    string
	s3Secret      *v1.Secret
	client        client.Client
	backupStorage *storageapi.BackupStorage
}

func NewBlob(ctx context.Context, c client.Client, bs *storageapi.BackupStorage) (*Blob, error) {
	switch bs.Spec.Storage.Provider {
	case storageapi.ProviderS3:
		return s3Blob(ctx, c, bs)
	case storageapi.ProviderGCS:
		return gcsBlob(ctx, c, bs)
	case storageapi.ProviderAzure:
		return azureBlob(ctx, c, bs)
	case storageapi.ProviderLocal:
		return localBlob(bs)
	default:
		return nil, fmt.Errorf("unknown provider: %s", bs.Spec.Storage.Provider)
	}
}

func s3Blob(ctx context.Context, c client.Client, bs *storageapi.BackupStorage) (*Blob, error) {
	var err error
	var secret *v1.Secret
	if bs.Spec.Storage.S3.SecretName != "" {
		secret, err = getStorageSecret(ctx, c, bs)
	}

	return &Blob{
		client:        c,
		backupStorage: bs,
		s3Secret:      secret,
		prefix:        bs.Spec.Storage.S3.Prefix,
	}, err
}

func gcsBlob(ctx context.Context, c client.Client, bs *storageapi.BackupStorage) (*Blob, error) {
	if bs.Spec.Storage.GCS.SecretName != "" {
		secret, err := getStorageSecret(ctx, c, bs)
		if err != nil {
			return nil, err
		}
		if err = setGcsCredentialsToEnv(secret); err != nil {
			return nil, err
		}
	}
	return &Blob{
		backupStorage: bs,
		prefix:        bs.Spec.Storage.GCS.Prefix,
		storageURL:    fmt.Sprintf("%s%s", gcsPrefix, bs.Spec.Storage.GCS.Bucket),
	}, nil
}

func azureBlob(ctx context.Context, c client.Client, bs *storageapi.BackupStorage) (*Blob, error) {
	if bs.Spec.Storage.Azure.StorageAccount == "" {
		return nil, fmt.Errorf("storageAccount is empty")
	}
	if err := os.Setenv(azureStorageAccount, bs.Spec.Storage.Azure.StorageAccount); err != nil {
		return nil, err
	}

	if bs.Spec.Storage.Azure.SecretName != "" {
		secret, err := getStorageSecret(ctx, c, bs)
		if err != nil {
			return nil, err
		}
		if err = setAzureCredentialsToEnv(secret); err != nil {
			return nil, err
		}
	}
	return &Blob{
		backupStorage: bs,
		prefix:        bs.Spec.Storage.Azure.Prefix,
		storageURL:    fmt.Sprintf("%s%s", azurePrefix, bs.Spec.Storage.Azure.Container),
	}, nil
}

func localBlob(bs *storageapi.BackupStorage) (*Blob, error) {
	return &Blob{
		backupStorage: bs,
		storageURL:    fmt.Sprintf("%s%s?no_tmp_dir=true", localPrefix, bs.Spec.Storage.Local.MountPath),
	}, nil
}

func getStorageSecret(ctx context.Context, c client.Client, bs *storageapi.BackupStorage) (*v1.Secret, error) {
	var secretName string
	switch bs.Spec.Storage.Provider {
	case storageapi.ProviderS3:
		secretName = bs.Spec.Storage.S3.SecretName
	case storageapi.ProviderGCS:
		secretName = bs.Spec.Storage.GCS.SecretName
	case storageapi.ProviderAzure:
		secretName = bs.Spec.Storage.Azure.SecretName
	default:
		return nil, fmt.Errorf("unknown provider: %s", bs.Spec.Storage.Provider)
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: bs.Namespace,
			Name:      secretName,
		},
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		return nil, err
	}
	return secret, nil
}

func setGcsCredentialsToEnv(secret *v1.Secret) error {
	if val, ok := secret.Data[googleServiceAccountJsonKey]; !ok {
		return fmt.Errorf("storage secret missing %s key", googleServiceAccountJsonKey)
	} else {
		filePath := path.Join(credentialsDir, googleServiceAccountJsonKey)
		if err := writeDataIntoFile(filePath, val); err != nil {
			return err
		}
		if err := os.Setenv(googleApplicationCredentials, filePath); err != nil {
			return err
		}
	}
	return nil
}

func setAzureCredentialsToEnv(secret *v1.Secret) error {
	if val, ok := secret.Data[azureAccountKey]; !ok {
		return fmt.Errorf("storage secret missing %s key", azureAccountKey)
	} else {
		if err := os.Setenv(azureStorageKey, string(val)); err != nil {
			return err
		}
	}
	return nil
}

func writeDataIntoFile(filePath string, val []byte) error {
	dir, _ := path.Split(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o777)
		if err != nil {
			return err
		}
	}
	if err := os.WriteFile(filePath, val, 0o755); err != nil {
		return err
	}

	return nil
}

func (b *Blob) Exists(ctx context.Context, filepath string) (bool, error) {
	dir, filename := path.Split(filepath)
	bucket, err := b.openBucket(ctx, dir)
	if err != nil {
		return false, err
	}
	defer closeBucket(ctx, bucket)
	return bucket.Exists(ctx, filename)
}

func (b *Blob) Get(ctx context.Context, filepath string) ([]byte, error) {
	dir, fileName := path.Split(filepath)
	bucket, err := b.openBucket(ctx, dir)
	if err != nil {
		return nil, err
	}
	defer closeBucket(ctx, bucket)
	r, err := bucket.NewReader(ctx, fileName, nil)
	if err != nil {
		return nil, err
	}
	defer func(r *blob.Reader) {
		closeErr := r.Close()
		if closeErr != nil {
			logger := log.FromContext(ctx)
			logger.Error(closeErr, "failed to close reader")
		}
	}(r)
	return io.ReadAll(r)
}

func (b *Blob) Upload(ctx context.Context, filepath string, data []byte, contentType string) error {
	dir, fileName := path.Split(filepath)
	bucket, err := b.openBucket(ctx, dir)
	if err != nil {
		return err
	}
	defer closeBucket(ctx, bucket)

	w, err := bucket.NewWriter(ctx, fileName, &blob.WriterOptions{
		ContentType:                 contentType,
		DisableContentTypeDetection: true,
	})
	if err != nil {
		return err
	}
	_, writeErr := w.Write(data)
	closeErr := w.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return closeErr
}

func (b *Blob) Debug(ctx context.Context, filepath string, data []byte, contentType string) error {
	dir, fileName := path.Split(filepath)
	bucket, err := b.openBucketWithDebug(ctx, dir, true)
	if err != nil {
		return err
	}

	defer closeBucket(ctx, bucket)

	klog.Infof("Uploading data to backend...")
	w, err := bucket.NewWriter(ctx, fileName, &blob.WriterOptions{
		ContentType:                 contentType,
		DisableContentTypeDetection: true,
	})
	if err != nil {
		return err
	}
	_, writeErr := w.Write(data)
	closeErr := w.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}

	klog.Infof("Cleaning up data from backend...")
	return bucket.Delete(ctx, fileName)
}

func (b *Blob) List(ctx context.Context, dir string) ([][]byte, error) {
	bucket, err := b.openBucket(ctx, dir)
	if err != nil {
		return nil, err
	}
	defer closeBucket(ctx, bucket)
	var objects [][]byte
	iter := bucket.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if checkIfObjectFile(obj) {
			fName := path.Join(dir, obj.Key)
			file, err := b.Get(ctx, fName)
			if err != nil {
				return nil, err
			}
			objects = append(objects, file)
		}
	}
	return objects, nil
}

// ListDirN depth = 0 → immediate children only.
func (b *Blob) ListDirN(ctx context.Context, dir string, depth ...int) ([][]byte, error) {
	bucket, err := b.openBucket(ctx, dir)
	if err != nil {
		return nil, err
	}
	defer closeBucket(ctx, bucket)

	maxDepth := 0
	if len(depth) > 0 {
		maxDepth = depth[0]
	}

	relPrefix := strings.TrimSuffix(dir, "/")
	if relPrefix != "" {
		relPrefix += "/"
	}

	var dirs [][]byte

	var walk func(prefix string, curDepth int) error
	walk = func(prefix string, curDepth int) error {
		iter := bucket.List(&blob.ListOptions{
			Prefix:    prefix,
			Delimiter: "/",
		})
		for {
			obj, err := iter.Next(ctx)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if obj.IsDir {
				dirs = append(dirs, []byte(obj.Key))
				if maxDepth < 0 || curDepth < maxDepth {
					if err := walk(obj.Key, curDepth+1); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	if err := walk(relPrefix, 0); err != nil {
		return nil, err
	}
	return dirs, nil
}

func (b *Blob) Delete(ctx context.Context, filepath string, isDir bool) error {
	if isDir {
		return b.deleteDir(ctx, filepath)
	}
	dir, filename := path.Split(filepath)
	bucket, err := b.openBucket(ctx, dir)
	if err != nil {
		return err
	}
	defer closeBucket(ctx, bucket)
	return bucket.Delete(ctx, filename)
}

func (b *Blob) deleteDir(ctx context.Context, dir string) error {
	bucket, err := b.openBucket(ctx, dir)
	if err != nil {
		return err
	}
	defer closeBucket(ctx, bucket)
	var deleteErrs []error
	iter := bucket.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		filePath := fmt.Sprintf("%s/%s", dir, obj.Key)
		err = b.Delete(ctx, filePath, false)
		if err != nil {
			deleteErrs = append(deleteErrs, err)
		}
	}
	return errors.NewAggregate(deleteErrs)
}

func checkIfObjectFile(obj *blob.ListObject) bool {
	if !obj.IsDir && len(obj.Key) > 0 && obj.Key[len(obj.Key)-1] != '/' {
		return true
	}
	return false
}

func (b *Blob) openBucket(ctx context.Context, dir string) (*blob.Bucket, error) {
	return b.openBucketWithDebug(ctx, dir, false)
}

func (b *Blob) openBucketWithDebug(ctx context.Context, dir string, debug bool) (*blob.Bucket, error) {
	var bucket *blob.Bucket
	var err error
	if b.backupStorage.Spec.Storage.Provider == storageapi.ProviderS3 {
		cfg, err := b.getS3Config(ctx, debug)
		if err != nil {
			return nil, err
		}
		bucket, err = s3blob.OpenBucketV2(ctx, s3.NewFromConfig(cfg, func(options *s3.Options) {
			options.UsePathStyle = true
		}), b.backupStorage.Spec.Storage.S3.Bucket, nil)
		if err != nil {
			return nil, err
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, b.storageURL)
		if err != nil {
			return nil, err
		}
	}

	suffix := strings.Trim(path.Join(b.prefix, dir), "/") + "/"
	if suffix == string(os.PathSeparator) {
		return bucket, nil
	}
	return blob.PrefixedBucket(bucket, suffix), nil
}

func closeBucket(ctx context.Context, bucket *blob.Bucket) {
	closeErr := bucket.Close()
	if closeErr != nil {
		logger := log.FromContext(ctx)
		logger.Error(closeErr, "failed to close bucket")
	}
}

func (b *Blob) getS3Config(ctx context.Context, debug bool) (aws2.Config, error) {
	var loadOptions []func(*config.LoadOptions) error
	if b.backupStorage.Spec.Storage.S3.SecretName != "" {
		if b.backupStorage.Spec.Storage.S3.Endpoint != "" {
			loadOptions = append(loadOptions, config.WithBaseEndpoint(b.backupStorage.Spec.Storage.S3.Endpoint))
		}
	}
	if b.backupStorage.Spec.Storage.S3.Region != "" {
		loadOptions = append(loadOptions, config.WithRegion(b.backupStorage.Spec.Storage.S3.Region))
	}

	if debug {
		loadOptions = append(loadOptions, config.WithClientLogMode(
			aws2.LogRetries|aws2.LogRequestWithBody|aws2.LogResponseWithBody))
	}

	if b.backupStorage.Spec.Storage.S3.SecretName != "" {
		id, ok := b.s3Secret.Data[awsAccessKeyId]
		if !ok {
			return aws2.Config{}, fmt.Errorf("storage secret %s/%s missing %s key", b.s3Secret.Namespace, b.s3Secret.Name, awsAccessKeyId)
		}
		key, ok := b.s3Secret.Data[awsSecretAccessKey]
		if !ok {
			return aws2.Config{}, fmt.Errorf("storage Secret %s/%s missing %s key", b.s3Secret.Namespace, b.s3Secret.Name, awsSecretAccessKey)
		}

		loadOptions = append(loadOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(string(id), string(key), ""),
		))

		needsTLS := b.backupStorage.Spec.Storage.S3.InsecureTLS || len(b.s3Secret.Data[caCertData]) > 0
		if needsTLS {
			httpClient, err := configureTLS(b.s3Secret.Data[caCertData],
				b.backupStorage.Spec.Storage.S3.InsecureTLS)
			if err != nil {
				return aws2.Config{}, err
			}
			loadOptions = append(loadOptions, config.WithHTTPClient(httpClient))
		}
	}

	return config.LoadDefaultConfig(ctx, loadOptions...)
}

func configureTLS(caCert []byte, insecureTLS bool) (*http.Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureTLS,
	}
	if len(caCert) > 0 {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}
	rt := http.DefaultTransport.(*http.Transport).Clone()
	rt.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: rt,
	}, nil
}

func (b *Blob) SetPathAsDir(ctx context.Context, path string) error {
	bucket, err := b.openBucket(ctx, path)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(path, "/") {
		path = fmt.Sprintf("%s/", path)
	}
	w, err := bucket.NewWriter(ctx, path, nil)
	if err != nil {
		return err
	}
	_, writeErr := w.Write([]byte(""))
	closeErr := w.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return closeErr
}
