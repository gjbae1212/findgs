# FindGS

<p align="left">
<!-- <a href="https://hits.seeyoufarm.com"/><img src="https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Fgjbae1212%2Ffindgs"/></a> -->
<a href="https://img.shields.io/badge/language-golang-blue"><img src="https://img.shields.io/badge/language-golang-blue" alt="language" /></a>
<a href="/LICENSE"><img src="https://img.shields.io/badge/license-MIT-GREEN.svg" alt="license" /></a>
</p>

**FindGS** searches **your starred github repositories** that matched your input text from README, Name, Topic, Description.

**Motivation**    

You or I have many starred repositories in github, because for using it in someday.   
With stacking more and more your starred repositories, you can **difficult** to find **wanted repositories**, because github doesn't support to search from such as **README**.      

**FindGS** is an interactive CLI using your github token for searching repositories.
> Notice that **FindGS** is using boltDB(cached) and bleve(indexing) internally.
> Because Github API is limited 5000 per hourly, so it's required DB for caching and for searching with higher performance.  
> And **FindGS** updates cached data with 1 hour interval when running it.

It's implemented using Golang.

<p align="center">
<img src="https://storage.googleapis.com/gjbae1212-asset/findgs/findgs_main.gif" width="900" height="600"/>
</p>
 
## Getting Started
### Prerequisite
### Install

## Features
## License
