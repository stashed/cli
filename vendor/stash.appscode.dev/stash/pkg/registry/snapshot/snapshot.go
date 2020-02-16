/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package snapshot

import (
	"context"

	"stash.appscode.dev/apimachinery/apis/repositories"
	repov1alpha1 "stash.appscode.dev/apimachinery/apis/repositories/v1alpha1"
	stash "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/util"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	restconfig "k8s.io/client-go/rest"
	core_util "kmodules.xyz/client-go/core/v1"
)

type REST struct {
	stashClient versioned.Interface
	kubeClient  kubernetes.Interface
	config      *restconfig.Config
}

var _ rest.Scoper = &REST{}
var _ rest.Getter = &REST{}
var _ rest.Lister = &REST{}
var _ rest.GracefulDeleter = &REST{}
var _ rest.GroupVersionKindProvider = &REST{}
var _ rest.CategoriesProvider = &REST{}

func NewREST(config *restconfig.Config) *REST {
	return &REST{
		stashClient: versioned.NewForConfigOrDie(config),
		kubeClient:  kubernetes.NewForConfigOrDie(config),
		config:      config,
	}
}

func (r *REST) NamespaceScoped() bool {
	return true
}

func (r *REST) New() runtime.Object {
	return &repositories.Snapshot{}
}

func (r *REST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return repov1alpha1.SchemeGroupVersion.WithKind(repov1alpha1.ResourceKindSnapshot)
}

func (r *REST) Categories() []string {
	return []string{"storage", "appscode", "all"}
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("missing namespace")
	}

	repoName, snapshotId, err := util.GetRepoNameAndSnapshotID(name)
	if err != nil {
		return nil, apierrors.NewBadRequest(err.Error())
	}

	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(repoName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, apierrors.NewNotFound(stash.Resource(stash.ResourceSingularRepository), repoName)
		} else {
			return nil, apierrors.NewInternalError(err)
		}
	}

	snapshots, err := r.GetVersionedSnapshots(repo, []string{snapshotId}, false)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	if len(snapshots) == 0 {
		return nil, apierrors.NewNotFound(repositories.Resource(repov1alpha1.ResourceSingularSnapshot), name)
	}

	// TODO: return &snapshots[0], nil
	snapshot := &snapshots[0]
	return snapshot, nil
}

func (r *REST) NewList() runtime.Object {
	return &repositories.SnapshotList{}
}

func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("missing namespace")
	}

	repos, err := r.stashClient.StashV1alpha1().Repositories(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	var selectedRepos []stash.Repository
	if options.LabelSelector != nil {
		for _, r := range repos.Items {
			repoLabels := map[string]string{
				"repository": r.Name,
			}
			if r.Labels != nil {
				repoLabels = core_util.UpsertMap(repoLabels, r.Labels)
			}
			if options.LabelSelector.Matches(labels.Set(repoLabels)) {
				selectedRepos = append(selectedRepos, r)
			}
		}
	} else {
		selectedRepos = repos.Items
	}

	snapshotList := &repositories.SnapshotList{
		Items: make([]repositories.Snapshot, 0),
	}
	for _, repo := range selectedRepos {
		snapshots, err := r.GetVersionedSnapshots(&repo, nil, false)
		if err != nil {
			return nil, apierrors.NewInternalError(err)
		}
		snapshotList.Items = append(snapshotList.Items, snapshots...)
	}

	// k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructured_list.go
	// unstructured.UnstructuredList{}
	return snapshotList, nil
}

func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, false, apierrors.NewBadRequest("missing namespace")
	}
	repoName, snapshotId, err := util.GetRepoNameAndSnapshotID(name)
	if err != nil {
		return nil, false, apierrors.NewBadRequest(err.Error())
	}
	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(repoName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(stash.Resource(stash.ResourceSingularRepository), repoName)
		} else {
			return nil, false, apierrors.NewInternalError(err)
		}
	}

	// first, check if the snapshot exist
	snapshots, err := r.GetVersionedSnapshots(repo, []string{snapshotId}, false)
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	} else if len(snapshots) == 0 {
		return nil, false, apierrors.NewNotFound(repositories.Resource(repov1alpha1.ResourceSingularSnapshot), name)
	}
	// delete snapshot
	if err = r.ForgetVersionedSnapshots(repo, []string{snapshotId}, false); err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}

	return nil, true, nil
}
