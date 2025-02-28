# DB Console

This directory contains the client-side code for CockroachDB's web-based DB
Console, which provides details about a cluster's performance and health. See the
[DB Console docs](https://www.cockroachlabs.com/docs/stable/ui-overview.html)
for an expanded overview.

## Getting Started

To start developing the UI, be sure you're able to build and run a CockroachDB
node. Instructions for this are located in the top-level README. Every Cockroach
node serves the UI, by default on port 8080, but you can customize the port with
the `--http-port` flag. If you've started a node with the default options,
you'll be able to access the UI at <http://localhost:8080>. If you've started
a node using `demo`, the default port is 8081 and you'll be able to access the UI
at <http://localhost:8081>.

Immdiately upon cloning this repo, your editor may report errors from
[eslint](https://eslint.org/) claiming that `@cockroachlabs/eslint-plugin-crbd`
failed to load. Solve this issue by running the linter once to build the
CRDB-specific plugin, which lives in this repo:
```shell
$ make ui-lint
```

or by building the plugin manually:
```shell
$ pushd pkg/ui/workspaces/eslint-plugin-crbd; yarn && yarn build; popd
```

Behind the scenes, our UI is compiled using a collection of tools that depends on
[Node.js](https://nodejs.org/) and are managed with
[Yarn](https://yarnpkg.com), a package manager that offers more deterministic
package installation than NPM. LTS versions of NodeJS and Yarn v1.x.x are known
to work. [Chrome](https://www.google.com/chrome/), Google's internet browser.
Unit tests are run using Chrome's "Headless" mode.

## Developing

When making changes to the UI, it is desirable to see those changes with data
from an existing cluster without rebuilding and relaunching the cluster for each
change. This is useful for rapidly visualizing local development changes against
a consistent and realistic dataset.

We've created a simple NodeJS proxy to accomplish this. This server serves all
requests for web resources (JavaScript, HTML, CSS) out of the code in this
directory, while proxying all API requests to the specified CockroachDB node.

To use this proxy, in Cockroach's root directory run:
```shell
$ ./dev ui watch --db=<target-cluster-http-uri>
```

then navigate to `http://localhost:3000` to access the UI.

To proxy to a cluster started up in secure mode, in Cockroach's root directory run:
```shell
$ ./dev ui watch --db=<target-cluster-http-uri> --secure
```

While the proxy is running, any changes you make in the `src` directory will
trigger an automatic recompilation of the UI. This recompilation should be much
faster than a cold compile—usually less than one second—as Webpack can reuse
in-memory compilation artifacts from the last compile.

Note that calling `./dev` above will hold the bazel lock, but only for the duration of the
initial build. The watcher itself releases the bazel lock, so it's perfectly reasonable to
run `./dev ui watch` in one shell and `./dev build` in another.


### Working with the `cluster-ui` dependency

Many page-level components have been extracted into a
separate repository for sharing with other applications.
You can read all about this division in the [README for the
package](https://github.com/cockroachdb/cockroach/blob/master/pkg/ui/workspaces/cluster-ui/README.md)
which describes a dev workflow that fits well with this package.

### Clearing the local cache
If the vendor directory becomes corrupted, clear it by running in the root Cockroach directory:
```shell
$ git submodule update --init --recursive
```

If the UI cache becomes corrupted, clear it with:
```shell
$ ./dev ui clean --all
$ make ui-maintainer-clean  # Deprecation soon.
```

If all else fails, run
```shell
bazel clean --expunge
```
though be warned your next build will take a while.

## CCL Build

In CCL builds, code in `pkg/ui/ccl/src` overrides code in `pkg/ui/src` at build
time, via a Webpack import resolution rule. E.g. if a file imports
`src/views/shared/components/licenseType`, it'll resolve to
`pkg/ui/src/views/shared/components/licenseType` in an OSS build, and
`pkg/ui/ccl/src/views/shared/components/licenseType` in a CCL build.

CCL code can import OSS code by prefixing paths with `oss/`, e.g.
`import "oss/src/myComponent"`. By convention, this is only done by a CCL file
importing the OSS version of itself, e.g. to render the OSS version of itself
when the trial period has expired.

## Running tests

To run the tests outside of CI:

```shell
$ ./dev ui test
```

## Managing dependencies

The NPM registry (and the Yarn proxy in front of it) have historically proven
quite flaky. Errors during `yarn install` were the leading cause of spurious CI
failures in the first half of 2018.

The solution is to check our JavaScript dependencies into version control, like
we do for our Go dependencies. Checking in the entire node_modules folder is a
non-starter: it clocks in at 230MB and 28k files at the time of writing.
Instead, we use a Yarn feature called the [offline mirror]. We ship a [.yarnrc]
file that instructs Yarn to save a tarball of each package we depend on in the
[yarn-vendor] folder. These tarballs are then checked in to version control. To
avoid cluttering the main repository, we've made the yarn-vendor folder a Git
submodule that points at the [cockroachdb/yarn-vendored] repository.

Be sure to commit modifications resulting from dependency changes, like
updates to `package.json` and `yarn.lock`.

### Adding a dependency

Let's pretend you want to add a dependency on `left-pad`. Just use `yarn add`
like you normally would:

```bash
$ cd $GOPATH/src/github.com/cockroachdb/cockroach/pkg/ui
$ yarn add FOO
```

When Yarn finishes, `git status` will report something like this:

```bash
$ git status
Changes not staged for commit:
	modified:   pkg/ui/package.json
	modified:   pkg/ui/yarn-vendor (untracked content)
	modified:   pkg/ui/yarn.lock
```

The changes to package.json and yarn.lock are the normal additions of the new
dependency information to the manifest files. The changes to yarn-vendor are
unique to the offline mirror mode. Let's look more closely:

```bash
$ git -C yarn-vendor status
Untracked files:
	left-pad-1.3.0.tgz
```

Yarn has left you a tarball of the new dependency. Perfect! If you were adding
a more complicated dependency, you'd likely see some transitive dependencies
appear as well.

The process from here is exactly the same as updating any of our other vendor
submodules. Make a new branch in the submodule, commit the new tarball, and push
it:

```bash
$ cd yarn-vendor
$ git checkout -b YOURNAME/add-left-pad
$ git add .
$ git commit -m 'Add left-pad@1.3.0'
$ git push origin add-left-pad
```

Be sure to push to [cockroachdb/yarn-vendored] directly instead of a personal
fork. Otherwise TeamCity won't be able to find the commit.

Then, return to the main repository and commit the changes, including the new
submodule commit. Push that and make a PR:

```bash
$ cd ..
$ git checkout -b add-left-pad
$ git add pkg/ui
$ git commit -m 'ui: use very smart approach to pad numbers with zeros'
$ git push YOUR-REMOTE add-left-pad
```

This time, be sure to push to your personal fork. Topic branches are not
permitted in the main repository.

When your PR has been approved, please be sure to merge your change to
yarn-vendored to master and delete your topic branch:

```bash
$ cd yarn-vendor
$ git checkout master
$ git merge add-left-pad
$ git push origin master
$ git push origin -d add-left-pad
```

This last step is extremely important! Any commit in yarn-vendored that is
referenced by the main repository must remain forever accessible, or it will be
impossible for future `git clone`s to build that version of Cockroach. GitHub
will garbage collect commits that are not accessible from any branch or tag, and
periodically, someone comes along and cleans up old topic branches in
yarn-vendored, potentially removing the only reference to a commit. By merging
your commit on master, you ensure that your commit will not go missing.

### Removing a dependency

To remove a dependency, just run `yarn remove`.

Note that removing a dependency will not remove its tarball from the yarn-vendor
folder. This is not as bad as it sounds. When Git fetches submodules, it always
performs a full clone, so it would wind up downloading deleted tarballs anyway
when it fetched older commits.

TODO(benesch): Yarn's offline mode has an additional option,
`yarn-offline-mirror-pruning`, that cleans up removed dependencies' tarballs.
Look into using this once [dcodeIO/protobuf.js#716] is resolved. At the moment,
ProtobufJS tries to install some dependencies at runtime by invoking `npm
install` itself (!); we avoid this by running `yarn install` on its behalf, but
this means two separate packages share the same yarn-vendor folder and this
confuses Yarn's pruning logic.

If the size of the yarn-vendored repository becomes problematic, we can look
into offloading the large files into something like [Git LFS]. This is
contingent upon resolving the above TODO.

[cockroachdb/yarn-vendored]: https://github.com/cockroachdb/yarn-vendored
[dcodeIO/protobuf.js#716]: https://github.com/dcodeIO/protobuf.js#716
[main app bundle]: ./webpack.config.js
[Git LFS]: https://git-lfs.github.com
[offline mirror]: https://yarnpkg.com/blog/2016/11/24/offline-mirror/
[.yarnrc]: ./yarnrc
[yarn-vendor]: ./yarn-vendor
