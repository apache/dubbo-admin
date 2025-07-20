#!/usr/bin/env bash

function msg_green() {
  builtin echo -en "\033[1;32m"
  echo "$@"
  builtin echo -en "\033[0m"
}

function msg_red() {
  builtin echo -en "\033[1;31m" >&2
  echo "$@" >&2
  builtin echo -en "\033[0m" >&2
}

function msg_yellow() {
  builtin echo -en "\033[1;33m"
  echo "$@"
  builtin echo -en "\033[0m"
}

function msg() {
  builtin echo -en "\033[1m"
  echo "$@"
  builtin echo -en "\033[0m"
}

function msg_err() {
  msg_red "$@"
  exit 1
}
