# FindGS

<p align="left">
<a href="https://hits.seeyoufarm.com"/><img src="https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Fgjbae1212%2Ffindgs"/></a>
<a href="https://img.shields.io/badge/language-golang-blue"><img src="https://img.shields.io/badge/language-golang-blue" alt="language" /></a>
<a href="/LICENSE"><img src="https://img.shields.io/badge/license-MIT-GREEN.svg" alt="license" /></a>
</p>

**FindGS** searches for **your starred repositories** in Github that are matched your input text to README, Name, Topic, Description.

**Motivation**  
Maybe you have many starred repositories in github for using it in someday.   
With stacking more and more your starred repositories, you can **difficult** to find **wanted repositories** in starred repositories.   
Because github site doesn't officially support to search for it in README.          

**FindGS** is an interactive CLI using your github token for searching repositories.
> Notice that **FindGS** makes internally caching db and indexing in local.
> Because Github API is limited 5000 per hourly, so it's required something for caching and for searching with higher performance.  
> And **FindGS** updates cached data an interval of 1 hour when running it.

It's implemented using **Golang**.
<br/> <br/>
<p align="center">
<img src="https://storage.googleapis.com/gjbae1212-asset/findgs/findgs_hello.gif"/>
</p>
<br/>

## Getting Started

### Prerequisite
It's required [**github personal access token**](https://github.com/settings/tokens).
<p align="center">
<img src="https://storage.googleapis.com/gjbae1212-asset/findgs/findgs_token.png"/>
</p>
<br/>

  
**This token should set global environment or pass to **findgs**.**
```bash
# EX1)
export GITHUB_TOKEN=your-token # .zshrc or .bash_profile 
findgs run

# EX2)
findgs run -t your-token 
```

### Install
Use to **Homebrew** if you want to install mac, but also you can download from [**releases**](https://github.com/gjbae1212/findgs/releases).
```bash
# mac 
$ brew tap gjbae1212/findgs
$ brew install findgs

# linux
$ wget https://github.com/gjbae1212/findgs/releases/download/v1.0.1/findgs_1.0.1_Linux_x86_64.tar.gz

# window
$ wget https://github.com/gjbae1212/findgs/releases/download/v1.0.1/findgs_1.0.1_Windows_x86_64.tar.gz
```

## Features
**FindGS** is currently to support the following features:
- ```findgs clear```
- ```findgs run```
#### findgs clear
Delete cached db and indexed data in local.
```bash
$ findgs clear
```
------
#### findgs run
Run an interactive CLI for searching your starred repositories in Github.
```bash
$ findgs run # need to `export GITHUB_TOKEN=your-token`

or 

$ finds run -t your-token 
```
 
**An interactive CLI** is currently to support the following commands: 
 
**search**  
`search` command searches your starred repositories using input text. Also it's to support wildcard searching.  
```bash  
>> search cli tool for aws or gcp
>> search hello* 
```  

**open**  
`open` command show your selected repository to browser.  
```bash
>> open name blahblah
```

**list**  
`list` command show recently searched result.
```bash
>> list
```

**score**  
`score` command sets a score that can search repositories equal to or higher than the score.( 0 <= score)
```bash
# default score 0.1
>> score 0.5 # change score to 0.5 
```

**exit**  
`exit` program.
```bash
>> exit 
```    
------

## License
This project is following The MIT.
