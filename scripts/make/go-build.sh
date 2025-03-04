#!/bin/sh

# AdGuard DNS Build Script
#
# The commentary in this file is written with the assumption that the reader
# only has superficial knowledge of the POSIX shell language and alike.
# Experienced readers may find it overly verbose.

# The default verbosity level is 0.  Show every command that is run and every
# package that is processed if the caller requested verbosity level greater than
# 0.  Also show subcommands if the requested verbosity level is greater than 1.
# Otherwise, do nothing.
verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '1' ]
then
	env
	set -x
	v_flags='-v=1'
	x_flags='-x=1'
elif [ "$verbose" -gt '0' ]
then
	set -x
	v_flags='-v=1'
	x_flags='-x=0'
else
	set +x
	v_flags='-v=0'
	x_flags='-x=0'
fi
readonly x_flags v_flags

# Exit the script if a pipeline fails (-e), prevent accidental filename
# expansion (-f), and consider undefined variables as errors (-u).
set -e -f -u

# Allow users to override the go command from environment.  For example, to
# build two releases with two different Go versions and test the difference.
go="${GO:-go}"
readonly go

# Set the build parameters unless already set.
branch="${BRANCH:-$( git rev-parse --abbrev-ref HEAD )}"
buildtime="${BUILD_TIME:-$( date -u +%FT%TZ%z )}"
revision="${REVISION:-$( git rev-parse --short HEAD )}"
version="${VERSION:-0}"
readonly branch buildtime revision version

# Compile them in.
version_pkg='github.com/AdguardTeam/AdGuardDNS/internal/agd'
ldflags="-s -w"
ldflags="${ldflags} -X ${version_pkg}.branch=${branch}"
ldflags="${ldflags} -X ${version_pkg}.buildtime=${buildtime}"
ldflags="${ldflags} -X ${version_pkg}.revision=${revision}"
ldflags="${ldflags} -X ${version_pkg}.version=${version}"
readonly ldflags version_pkg

# Allow users to limit the build's parallelism.
parallelism="${PARALLELISM:-}"
readonly parallelism

# Use GOFLAGS for -p, because -p=0 simply disables the build instead of leaving
# the default value.
if [ "${parallelism}" != '' ]
then
        GOFLAGS="${GOFLAGS:-} -p=${parallelism}"
fi
readonly GOFLAGS
export GOFLAGS

# Allow users to specify a different output name.
out="${OUT:-AdGuardDNS}"
readonly out

o_flags="-o=${out}"
readonly o_flags

# Allow users to enable the race detector.  Unfortunately, that means that cgo
# must be enabled.
if [ "${RACE:-0}" -eq '0' ]
then
	cgo_enabled='0'
	race_flags='--race=0'
else
	cgo_enabled='1'
	race_flags='--race=1'
fi
readonly cgo_enabled race_flags

if [ "$verbose" -gt '0' ]
then
	"$go" env
fi

"$go" build --ldflags="$ldflags" "$race_flags" --trimpath "$o_flags" "$v_flags" "$x_flags"
