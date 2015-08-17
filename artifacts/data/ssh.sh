#!/bin/sh
exec ssh -o StrictHostKeyChecking=no -i deploy_key "$@"
