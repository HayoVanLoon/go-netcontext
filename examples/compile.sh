#!/usr/bin/env bash

OLD_DIR="${PWD}"

REPO_PROVIDER="github.com"
REPO_ACCCOUNT="acme"
GENPROTO_MODULE="${REPO_PROVIDER}/${REPO_ACCCOUNT}/go-genproto"
# The module's subdirectory for GAPIC client packages
CLIENT_DIR=clients

set -eo pipefail

# Default to current directory
SRC=.
TARGET=
OUT=
FORCE=
INCLUDES=
NO_DESCRIPTOR=
NO_MOD=
NO_GATEWAY=
PROTO_GOOGLEAPIS=${PROTO_GOOGLEAPIS}

usage() {
	echo "Usage
  $0 [--source-dir <path>] --out <path> [--target <path>] [--includes path] [--api-common-protos <path>] [--no-descriptor] [--no-mod] [--force]

Parameters

  --source-dir, -i     proto directory
  --output-dir, -o     relative path to output directory for generated code
  --target             starting from the source directory, use only this sub directory
  --include            path to other proto directories to include, divided by spaces
  --api-common-protos  specify api-common-protos directory (if not set via \$PROTO_GOOGLEAPIS)
  --no-descriptor      do not generate API descriptor
  --no-mod             do not generate a module, nor generate clients
  --force              overwrite existing (generated) code

Examples

  # Read from current directory, write to ./lib.
  
  ${0} -o lib


  # Read from ./proto, write to ./lib
  
  ${0} -i proto -o lib
"
}

while true; do
	case "${1}" in
	--source-dir | -i)
		SRC="${2}"
		shift 2
		;;
	--out | -o)
		OUT="${2}"
		shift 2
		;;
	--target | -t)
		TARGET="${2}"
		shift 2
		;;
	--api-common-protos)
		PROTO_GOOGLEAPIS="${2}"
		shift 2
		;;
	--includes)
		INCLUDES="${2}"
		shift 2
		;;
	--no-descriptor)
		NO_DESCRIPTOR=1
		shift 1
		;;
  --no-gateway)
    NO_GATEWAY=1
    shift 1
    ;;
	--no-mod)
		NO_MOD=1
		shift 1
		;;
	--force | -f)
		FORCE=1
		shift 1
		;;
	*)
		if [ -n "${1}" ]; then
			usage
			echo
			echo "Unexpected parameter ${1}"
			exit 3
		fi
		break
		;;
	esac
done

set -eou pipefail

# Path to api-common-protos. PROTO_GOOGLEAPIS is used for historical reasons.
if [ -z "${PROTO_GOOGLEAPIS}" ]; then
	usage
	echo >&2 "PROTO_GOOGLEAPIS has not been set"
	exit 3
fi

if [ -z "${OUT}" ]; then
	usage
	echo >&2 "Missing -o <output directory>"
	exit 3
fi

if [ -z "${SRC}" ]; then
	usage
	echo >&2 "Missing -i <source directory>"
	exit 3
fi

_OK=
ls "${OUT}" &>/dev/null && _OK=1
if [ "${_OK}" ] && [ -z "${FORCE}" ]; then
	usage
	echo >&2 "Output directory is not empty"
	exit 3
fi

cd "${SRC}"
SRC=.
OUT="${OLD_DIR}/${OUT}"

_INCLUDES=
if [ -n "${INCLUDES}" ]; then
  for i in ${INCLUDES}; do
    ls "$i" 1> /dev/null
    _INCLUDES=${_INCLUDES}\ -I"${i}"
  done
fi

compile() {
  _OK=
	echo "Compiling descriptor, protobuf and gRPC code ..."

	FILES=$(find "${SRC}/${TARGET}" -type f -name "*.proto" | sort)
	if [ -z "${FILES}" ]; then
		echo >&2 "Could not find any proto files starting with ${SRC}/"
		exit 3
	fi

  _DESCRIPTOR="--descriptor_set_out="${OUT}/descriptor.pb" --include_imports --include_source_info"
  if [ -n "${NO_DESCRIPTOR}" ]; then
    _DESCRIPTOR=
  fi
  _SOURCE_RELATIVE=
  if [ -n "${NO_MOD}" ]; then
    _SOURCE_RELATIVE="--go_opt=paths=source_relative --go-grpc_opt=paths=source_relative"
  fi

  _GATEWAY_OPTS=
  if [ -z "${NO_GATEWAY}" ]; then
    _GATEWAY_OPTS="--grpc-gateway_out ${OUT} --grpc-gateway_opt logtostderr=true"
    _SOURCE_RELATIVE="${_SOURCE_RELATIVE} --grpc-gateway_opt=paths=source_relative"
  fi

	mkdir -p "${OUT}"

	protoc \
		${_DESCRIPTOR} \
		--go_out="${OUT}" \
		--go-grpc_out="${OUT}" \
		${_SOURCE_RELATIVE} ${_GATEWAY_OPTS}\
		-I"${PROTO_GOOGLEAPIS}" \
		${_INCLUDES}\
		-I"${SRC}" \
		${FILES}

	report "${OUT}"
  _OK=1
}

protoc_gapic() {
  _OK=
	# use output to prevent 'parsing' protos for api annotations
	GPRC_OUTPUT=$(find "${OUT}" -type f -name "*_grpc.pb.go" | sort)
	if [ -z "${GPRC_OUTPUT}" ]; then
		echo >&2 "Could not find any generated grpc files starting with ${OUT}"
		exit 3
	fi

	for file in ${GPRC_OUTPUT}; do
		# Escape slashes
		safeOUT="$(sed 's/\//\\\//g' <<<"${OUT}")"
		SERVICE_DIR="$(sed -E "s/^${safeOUT}\/(.+)\/\w+_grpc\.pb\.go/\1/g" <<<"${file}")"
		PACKAGE="$(sed -E "s/.+\/([^/]+)\/v.+$/\1/g" <<<"${SERVICE_DIR}")"
		PACKAGE_IMPORT="$(sed -E "s/(.+)(\/[^/]+\/v.+$)/\1\/${CLIENT_DIR}\2/g" <<<"${SERVICE_DIR}")"

		# Skip health check service
		if [ "${PACKAGE}" = "health" ]; then
			continue
		fi

		echo "Generating GAPIC client for ${PACKAGE} ..."
		SRC_DIR="${SRC}/${PROTO_SERVICE_DIRS}/${PACKAGE}"
		ls "${SRC_DIR}" 1> /dev/null
		FILES=$(find "${SRC_DIR}" -type f -name "*.proto" | sort)
		if [ -z "${FILES}" ]; then
			echo >&2 "Could not find any proto files in ${SRC_DIR}"
			exit 3
		fi

		CONFIG_PARAM=
		CONFIG=$(find "${SRC}/${PROTO_SERVICE_DIRS}/${PACKAGE}" -type f -name "${PACKAGE}.json" | sort)
		if [ -n "${CONFIG}" ]; then
			CONFIG_PARAM=--go_gapic_opt=grpc-service-config="${CONFIG}"
		else
			echo "No GAPIC config for service ${PACKAGE}"
		fi

		protoc \
			--go_gapic_out="${OUT}" \
			--go_gapic_opt=go-gapic-package="${PACKAGE_IMPORT};${PACKAGE}" \
			${CONFIG_PARAM} \
			-I"${PROTO_GOOGLEAPIS}" \
			${_INCLUDES}\
			-I"${SRC}" \
			${FILES}
	done || exit 3

	report "${OUT}/${GENPROTO_MODULE}/${CLIENT_DIR}"
	_OK=1
}

init_mods() {
  _OK=
	# use subshell to freely change directories (and back)
	(
		echo "Initialising module ..."
		cd "${OUT}/${GENPROTO_MODULE}"
		go mod init "${GENPROTO_MODULE}" 2>/dev/null || echo "Already exists, continuing with tidying ..."
		go mod tidy 2>/dev/null
		echo "... done."
	)
  _OK=1
}

report() {
	COUNT=$(find "${1}" -type f | wc -l)
	echo "Produced ${COUNT} file(s)."
	echo
}

compile || cd "${OLD_DIR}"
if [ -z "${_OK}" ]; then
  echo ERROR!
  exit 3
fi

if [ -z "${NO_MOD}" ]; then
  protoc_gapic || cd "${OLD_DIR}"
  if [ -z "${_OK}" ]; then
    echo ERROR!
    exit 3
  fi

  init_mods || cd "${OLD_DIR}"
  if [ -z "${_OK}" ]; then
    echo ERROR!
    exit 3
  fi
else
  echo "Skipping module and client generation."
  echo
fi

cd "${OLD_DIR}"

echo Done!
echo
