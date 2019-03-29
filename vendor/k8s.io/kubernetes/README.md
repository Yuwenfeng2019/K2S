# Kubernetes

<img src="https://github.com/kubernetes/kubernetes/raw/master/logo/logo.png" width="100">

----

Kubernetes without the features I don't care about.

Rebase Instructions
-------------------

## Patch rebase

These are instructions for rebasing a patch version. For example if the current
K2S k8s version is v1.13.3 and v1.13.4 comes out these are the procedures on how
to rebase and create a new release.  If v1.14 comes out that procedure is different.

The below instructions will use the example of rebasing from v1.13.3 to v1.13.4.
For git commands the remote `rancher` is github.com/rancher/K2S and the remote
`upstream` refers to github.com/kubernetes/kubernetes

* Create a branch in github.com/rancher/K2S called K2S-${VERSION} that is branched
   from the upstream tag ${VERSION}.
   
```bash
VERSION=v1.13.4
git fetch upstream
git checkout -b K2S-${VERSION} ${VERSION}
git push rancher K2S-${VERSION}
```

* Start rebase
```bash
OLD_VERSION=v1.13.3
VERSION=v1.13.4
git fetch rancher
git checkout K2S-${VERSION}
git reset --hard rancher/K2S-${OLD_VERSION}
git rebase -i ${VERSION}
```
* When presented with the patch edit screen you want to drop an commit titled
   "Update Vendor" or a version commit like "v1.13.3-K2S.6"
* Continue rebase and resolve any conflicts.
* Run the below to update vendor and apply tag

```bash
VERSION=v1.13.4
./deps.sh && ./tag.sh ${VERSION}-k2s.1
```

* Update the README.md with anything that might have changed in the procedure
* Put in PR to github.com/rancher/K2S K2S-${VERSION} branch
* After merge apply ${VERSION}-K2S.1 tag in github then vendor into K2S
