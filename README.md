# Briefmail

[![Build Status](https://travis-ci.org/lukasdietrich/briefmail.svg?branch=master)](https://travis-ci.org/lukasdietrich/briefmail) [![Go Report Card](https://goreportcard.com/badge/github.com/lukasdietrich/briefmail)](https://goreportcard.com/report/github.com/lukasdietrich/briefmail) [![Docker Pulls](https://img.shields.io/docker/pulls/lukd/briefmail.svg)](https://hub.docker.com/r/lukd/briefmail)

Briefmail aims to be an easy to set up and self contained mail server for both
sending and receiving emails.

## Motivation

Receiving emails on your own domain means either paying for an email service or
setting up your own. The latter tends to require quite a bit of research in an
area you did not even know exists. What is *MTA*, *MUA* or *MDA*?

Briefmail **tries** to do most of the heavy lifting with reasonable defaults
and an intuitive configuration for simple tasks. Setting up a server to forward
emails to another address should not require hours of fiddling around trying to
understand configuration files.

Briefmail **does not try** to be a production ready, high performance system
for thousands of mailboxes configurable through a fancy interface. If you need
production performance, you will have to look after battle tested software.

## License

```
Copyright (C) 2018  Lukas Dietrich <lukas@lukasdietrich.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```
