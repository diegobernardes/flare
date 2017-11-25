# Contributing

## Prerequisites
1. Install Docker. 
2. [Install Go][go-install]. I'm working with 1.9+, it may work with early versions, but not granted.
3. Download the sources and switch the working directory:
    ```bash
    go get -u -d github.com/diegobernardes/flare
    cd $GOPATH/src/github.com/diegobernardes/flare
    ```
4. Configure the repository.
5. Run `make configure`.

## Working with Git and Go
First, you need to [fork][fork] the Flare repository. After this, add a new remote pointing to your fork.
```bash
git remote rename origin upstream
git remote add origin git@github.com:{you}/flare.git  # you may use http, your choice.
```

Now you can work with your own repository and fetch the changes from official Flare repository. Let's create a branch to work:
```bash
git checkout master
git checkout -t -b feature
# do the commits.
git push origin feature
```

We work with rebases to squash multiple commits on feature branchs, if you did more then 1 commit on a feature branch, you must do this:
```bash
git checkout feature
git rebase -i $(git merge-base feature master)
git push --force-with-lease
```

One last thing, before create the pull request, make sure your branch is updated:
```bash
git fetch upstream

git checkout master
git rebase upstream/master
git push --force-with-lease

git checkout feature
git rebase master
git push --force-with-lease
```

Now you ready to go, submit the pull request.

## Git commit messages
We follow a similar pattern used by Golang. I try to keep the first line lower then 72 characters and my hard limit is 100.
```
module: message

body

(Issue, Close, Resolves): #123
```

The module is something that represent the commit intent.


## Submitting a Pull Request
A typical workflow is:

1. Fork the repository.
2. Create a feature branch.
3. Add tests for your change.
4. Run `make pre-pr`. If your tests don't pass or the linter complain, return to step 3.
5. [Add, commit and push your changes.][git-help]
6. [Submit a pull request.][pull-req]

[go-install]: https://golang.org/doc/install
[fork]: https://help.github.com/articles/fork-a-repo
[git-help]: https://guides.github.com
[pull-req]: https://help.github.com/articles/using-pull-requests
