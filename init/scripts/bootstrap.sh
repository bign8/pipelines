#!/bin/bash

# CONFIGURE GIT
# TODO: install git
git config --global color.ui true
git config --global user.name "Nate Woods"
git config --global user.email big.nate.w@gmail.com

# Clone Repository
git clone https://bign8@github.com/bign8/pipelines.git
cd pipelines

# Setup console
echo "source ~/pipelines/config/dot.profile" > ~/.profile
source ~/.profile
