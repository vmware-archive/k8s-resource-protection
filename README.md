## resource protection

Resource protection project includes following components:

- allowed operations webhook: webhook looks for `rp.k14s.io/allowed-operations` annotation to control which operations are allowed against a resource. Possible values: `""` disallows all actions, to allow specific actions, use: `CREATE`, `UPDATE`, `DELETE`, `CONNECT`. Values are comma separated (e.g. `UPDATE,DELETE`). See [examples/locked-cm.yml](examples/locked-cm.yml). Related upstream Kubernetes issue: [https://github.com/kubernetes/kubernetes/issues/10179](https://github.com/kubernetes/kubernetes/issues/10179).

## Building

```
./hack/build.sh
```

To deploy:

```
# Install certmanager
ytt -f config-certmanager/config.yml --file-mark config.yml:type=yaml-plain \
  -f config-certmanager/patches.yml | kapp deploy -a certmgr -f- -y

# Install rp
ytt -f config/ | kbld -f- | kapp deploy -a rp -f- -c -y
```
