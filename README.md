# git-clean
Cleans up git repositores.

I always found myself with lots of old git branches that are no longer in use. `git-clean` is a very simple script the local git repo by running `git remote prune origin` and then removing all local branches that have been merged into master.

# Potential future features
- Clean branches on remote repos
