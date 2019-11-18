#!/bin/bash
set -e

if [ ! -z "${OLARIS_UID}" ]; then
  if [ ! "$(id -u olaris)" -eq "${OLARIS_UID}" ]; then
    # Change the UID
    usermod -o -u "${OLARIS_UID}" olaris
  fi
fi

if [ ! -z "${OLARIS_GID}" ]; then
  if [ ! "$(id -g olaris)" -eq "${OLARIS_GID}" ]; then
    groupmod -o -g "${OLARIS_GID}" olaris
  fi
fi

args=( "$@" )
# Login shell to properly set env vars
exec sudo -u olaris "$@"

