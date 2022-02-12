#! /bin/bash
eval $(egrep -v '^#' ../../.env | xargs) $1