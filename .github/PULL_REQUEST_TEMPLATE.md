The branch of 0.2.0 is master-0.2.0. If your PR is to solve the problem of 0.2.0, please submit the PR to it.
Please do not create a Pull Request without creating an issue first.

## What is the purpose of the change

XXXXX

## Brief changelog

XX

## Verifying this change

XXXX

Follow this checklist to help us incorporate your contribution quickly and easily:

* [ ] Make sure there is a Github issue filed for the change (usually before you start working on it). Trivial changes like typos do not require a Github issue. Your pull request should address just this issue, without pulling in other changes - one PR resolves one issue.
* [ ] Format the pull request title like `[ISSUE #123] Fix UnknownException when host config not exist`. Each commit in the pull request should have a meaningful subject line and body.
* [ ] Write a pull request description that is detailed enough to understand what the pull request does, how, and why.
* [ ] Write necessary unit-test to verify your logic correction, more mock a little better when cross module dependency exist.
* [ ] Run `mvn clean compile --batch-mode -DskipTests=false -Dcheckstyle.skip=false -Drat.skip=false -Dmaven.javadoc.skip=true to make sure basic checks pass.
