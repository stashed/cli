# Download command: copying snapshots from a pod

`pkg/download.go` (around line 287) currently uses `kubectl cp` to copy the downloaded snapshots from the repository pod into the local download directory. A stronger alternative is to stream a **compressed tar archive** over `kubectl exec`.

## Stream a compressed tar archive (recommended)

**Command shape**
```
kubectl exec -n <ns> -c <container> <pod> -- \
  tar -C <pod-dir> -czf - . | tar -C <local-dir> -xzf -
```

**Why this is better**
- **Efficient transfer**: gzip compression can significantly reduce transfer size for text-heavy or sparse data.
- **Fewer `kubectl cp` pitfalls**: avoids known `kubectl cp` issues with large directories, symlinks, or path edge cases.
- **Single stream**: one stdout stream from the pod, one extract locally—simple to reason about and retry.

**Operational notes**
- **Requires `tar` + `gzip`** in the container and locally.
- **CPU vs bandwidth tradeoff**: expect higher CPU usage during compression/decompression; for very large binary blobs, compression gains may be small.
- **Permissions**: run as a user that can read the snapshot directory and write to the local destination.

**Implementation sketch (Go)**
1. Build the pod-side command: `tar -C <pod-dir> -czf - .`
2. Execute via `kubectl exec` (or client-go exec) and stream stdout.
3. Pipe stdout to a local `tar -C <local-dir> -xzf -` process.
4. Surface stderr from both sides to logs for troubleshooting.


Woking Locally:

```bash
➤ kubectl exec -n demo -c stash stash-netvol-accessor-6dc79f6569-vlrmn -- \
        tar -C /tmp/destination/snapshots -czf - . \
      | tar -C ~/archiver -xzf -
anisur@anisur-rahman-PC:~/g/s/s/cli|ornl-issue⚡*?
➤ ls ~/archiver/
8ee3684c/

```