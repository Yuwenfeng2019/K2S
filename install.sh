#!/bin/sh
set -e

# Usage:
#   curl ... | ENV_VAR=... sh -
#       or
#   ENV_VAR=... ./install.sh
#
# Example:
#   Installing a server without traefik:
#     curl ... | INSTALL_K2S_EXEC="--no-deploy=traefik" sh -
#   Installing an agent to point at a server:
#     curl ... | K2S_TOKEN=xxx K2S_URL=https://server-url:6443 sh - 
#
# Environment variables:
#   - K2S_*
#     Environment variables which begin with K2S_ will be preserved for the
#     systemd service to use. Setting K2S_URL without explicitly setting
#     a systemd exec command will default the command to "agent", and we
#     enforce that K2S_TOKEN or K2S_CLUSTER_SECRET is also set.
#
#   - INSTALL_K2S_SKIP_DOWNLOAD
#     If set to true will not download K2S hash or binary.
#
#   - INSTALL_K2S_SYMLINK
#     If set to 'skip' will not create symlinks, 'force' will overwrite,
#     default will symlink if command does not exist in path.
#
#   - INSTALL_K2S_SKIP_ENABLE
#     If set to true will not enable or start k2s service.
#
#   - INSTALL_K2S_SKIP_START
#     If set to true will not start k2s service.
#
#   - INSTALL_K2S_VERSION
#     Version of k2s to download from github. Will attempt to download the
#     latest version if not specified.
#
#   - INSTALL_K2S_COMMIT
#     Commit of k2s to download from temporary cloud storage.
#     * (for developer & QA use)
#
#   - INSTALL_K2S_BIN_DIR
#     Directory to install k2s binary, links, and uninstall script to, or use
#     /usr/local/bin as the default
#
#   - INSTALL_K2S_BIN_DIR_READ_ONLY
#     If set to true will not write files to INSTALL_K2S_BIN_DIR, forces
#     setting INSTALL_K2S_SKIP_DOWNLOAD=true
#
#   - INSTALL_K2S_SYSTEMD_DIR
#     Directory to install systemd service and environment files to, or use
#     /etc/systemd/system as the default
#
#   - INSTALL_K2S_EXEC or script arguments
#     Command with flags to use for launching K2S in the systemd service, if
#     the command is not specified will default to "agent" if K2S_URL is set
#     or "server" if not. The final systemd command resolves to a combination
#     of EXEC and script args ($@).
#
#     The following commands result in the same behavior:
#       curl ... | INSTALL_K2S_EXEC="--no-deploy=traefik" sh -s -
#       curl ... | INSTALL_K2S_EXEC="server --no-deploy=traefik" sh -s -
#       curl ... | INSTALL_K2S_EXEC="server" sh -s - --no-deploy=traefik
#       curl ... | sh -s - server --no-deploy=traefik
#       curl ... | sh -s - --no-deploy=traefik
#
#   - INSTALL_K2S_NAME
#     Name of systemd service to create, will default from the K2S exec command
#     if not specified. If specified the name will be prefixed with 'K2S-'.
#
#   - INSTALL_K2S_TYPE
#     Type of systemd service to create, will default from the K2S exec command
#     if not specified.

GITHUB_URL=https://github.com/Yuwenfeng2019/K2S/releases
STORAGE_URL=https://storage.googleapis.com/k2s-ci-builds
DOWNLOADER=

# --- helper functions for logs ---
info()
{
    echo '[INFO] ' "$@"
}
fatal()
{
    echo '[ERROR] ' "$@" >&2
    exit 1
}

# --- fatal if no systemd or openrc ---
verify_system() {
    if [ -x /sbin/openrc-run ]; then
        HAS_OPENRC=true
        return
    fi
    if [ -d /run/systemd ]; then
        HAS_SYSTEMD=true
        return
    fi
    fatal "Can not find systemd or openrc to use as a process supervisor for k2s"
}

# --- add quotes to command arguments ---
quote() {
    for arg in "$@"; do
        printf '%s\n' "$arg" | sed "s/'/'\\\\''/g;1s/^/'/;\$s/\$/'/"
    done
}

# --- add indentation and trailing slash to quoted args ---
quote_indent() {
    printf ' \\\n'
    for arg in "$@"; do
        printf '\t%s \\\n' "$(quote "$arg")"
    done
}

# --- escape most punctuation characters, except quotes, forward slash, and space ---
escape() {
    printf '%s' "$@" | sed -e 's/\([][!#$%&()*;<=>?\_`{|}]\)/\\\1/g;'
}

# --- escape double quotes ---
escape_dq() {
    printf '%s' "$@" | sed -e 's/"/\\"/g'
}

# --- define needed environment variables ---
setup_env() {
    # --- use command args if passed or create default ---
    case "$1" in
        # --- if we only have flags discover if command should be server or agent ---
        (-*|"")
            if [ -z "${K2S_URL}" ]; then
                CMD_K2S=server
            else
                if [ -z "${K2S_TOKEN}" ] && [ -z "${K2S_CLUSTER_SECRET}" ]; then
                    fatal "Defaulted K2S exec command to 'agent' because K2S_URL is defined, but K2S_TOKEN or K2S_CLUSTER_SECRET is not defined."
                fi
                CMD_K2S=agent
            fi
        ;;
        # --- command is provided ---
        (*)
            CMD_K2S=$1
            shift
        ;;
    esac
    CMD_K2S_EXEC="${CMD_K2S}$(quote_indent "$@")"

    # --- use systemd name if defined or create default ---
    if [ -n "${INSTALL_K2S_NAME}" ]; then
        SYSTEM_NAME=k2s-${INSTALL_K2S_NAME}
    else
        if [ "${CMD_K2S}" = "server" ]; then
            SYSTEM_NAME=k2s
        else
            SYSTEM_NAME=k2s-${CMD_K2S}
        fi
    fi

    # --- check for invalid characters in system name ---
    valid_chars=$(printf '%s' "${SYSTEM_NAME}" | sed -e 's/[][!#$%&()*;<=>?\_`{|}/[:space:]]/^/g;' )
    if [ "${SYSTEM_NAME}" != "${valid_chars}"  ]; then
        invalid_chars=$(printf '%s' "${valid_chars}" | sed -e 's/[^^]/ /g')
        fatal "Invalid characters for system name:
            ${SYSTEM_NAME}
            ${invalid_chars}"
    fi

    # --- use sudo if we are not already root ---
    SUDO=sudo
    if [ `id -u` -eq 0 ]; then
        SUDO=
    fi

    # --- use systemd type if defined or create default ---
    if [ -n "${INSTALL_K2S_TYPE}" ]; then
        SYSTEMD_TYPE="${INSTALL_K2S_TYPE}"
    else
        if [ "${CMD_K2S}" = "server" ]; then
            SYSTEMD_TYPE=notify
        else
            SYSTEMD_TYPE=exec
        fi
    fi

    # --- use binary install directory if defined or create default ---
    if [ -n "${INSTALL_K2S_BIN_DIR}" ]; then
        BIN_DIR="${INSTALL_K2S_BIN_DIR}"
    else
        BIN_DIR=/usr/local/bin
    fi

    # --- use systemd directory if defined or create default ---
    if [ -n "${INSTALL_K2S_SYSTEMD_DIR}" ]; then
        SYSTEMD_DIR="${INSTALL_K2S_SYSTEMD_DIR}"
    else
        SYSTEMD_DIR=/etc/systemd/system
    fi

    # --- set related files from system name ---
    SERVICE_K2S=${SYSTEM_NAME}.service
    UNINSTALL_K2S_SH=${UNINSTALL_K2S_SH:-${BIN_DIR}/${SYSTEM_NAME}-uninstall.sh}
    KILLALL_K2S_SH=${KILLALL_K2S_SH:-${BIN_DIR}/k2s-killall.sh}

    # --- use service or environment location depending on systemd/openrc ---
    if [ "${HAS_SYSTEMD}" = "true" ]; then
        FILE_K2S_SERVICE=${SYSTEMD_DIR}/${SERVICE_K2S}
        FILE_K2S_ENV=${SYSTEMD_DIR}/${SERVICE_K2S}.env
    elif [ "${HAS_OPENRC}" = "true" ]; then
        $SUDO mkdir -p /etc/Yuwenfeng2019/K2S
        FILE_K2S_SERVICE=/etc/init.d/${SYSTEM_NAME}
        FILE_K2S_ENV=/etc/Yuwenfeng2019/K2S/${SYSTEM_NAME}.env
    fi

    # --- get hash of config & exec for currently installed k2s ---
    PRE_INSTALL_HASHES=$(get_installed_hashes)

    # --- if bin directory is read only skip download ---
    if [ "${INSTALL_K2S_BIN_DIR_READ_ONLY}" = "true" ]; then
        INSTALL_K2S_SKIP_DOWNLOAD=true
    fi
}

# --- check if skip download environment variable set ---
can_skip_download() {
    if [ "${INSTALL_K2S_SKIP_DOWNLOAD}" != "true" ]; then
        return 1
    fi
}

# --- verify an executabe K2S binary is installed ---
verify_K2S_is_executable() {
    if [ ! -x ${BIN_DIR}/K2S ]; then
        fatal "Executable K2S binary not found at ${BIN_DIR}/K2S"
    fi
}

# --- set arch and suffix, fatal if architecture not supported ---
setup_verify_arch() {
    if [ -z "$ARCH" ]; then
        ARCH=$(uname -m)
    fi
    case $ARCH in
        arm64)
            ARCH=arm64
            SUFFIX=-${ARCH}
            ;;
        aarch64)
            ARCH=arm64
            SUFFIX=-${ARCH}
            ;;
            fatal "Unsupported architecture $ARCH"
    esac
}

# --- verify existence of network downloader executable ---
verify_downloader() {
    # Return failure if it doesn't exist or is no executable
    [ -x "$(which $1)" ] || return 1

    # Set verified executable as our downloader program and return success
    DOWNLOADER=$1
    return 0
}

# --- verify existence of semanage when SELinux is enabled ---
verify_semanage() {
    if [ -x "$(which getenforce)" ]; then
        if [ "Disabled" != $(getenforce) ] && [ ! -x "$(which semanage)" ]; then
            fatal 'SELinux is enabled but semanage is not found'
        fi
    fi
}

# --- create tempory directory and cleanup when done ---
setup_tmp() {
    TMP_DIR=$(mktemp -d -t k2s-install.XXXXXXXXXX)
    TMP_HASH=${TMP_DIR}/k2s.hash
    TMP_BIN=${TMP_DIR}/k2s.bin
    cleanup() {
        code=$?
        set +e
        trap - EXIT
        rm -rf ${TMP_DIR}
        exit $code
    }
    trap cleanup INT EXIT
}

# --- use desired K2S version if defined or find latest ---
get_release_version() {
    if [ -n "${INSTALL_K2S_COMMIT}" ]; then
        VERSION_K2S="commit ${INSTALL_K2S_COMMIT}"
    elif [ -n "${INSTALL_K2S_VERSION}" ]; then
        VERSION_K2S=${INSTALL_K2S_VERSION}
    else
        info "Finding latest release"
        case $DOWNLOADER in
            curl)
                VERSION_K2S=$(curl -w '%{url_effective}' -I -L -s -S ${GITHUB_URL}/latest -o /dev/null | sed -e 's|.*/||')
                ;;
            wget)
                VERSION_K2S=$(wget -SqO /dev/null ${GITHUB_URL}/latest 2>&1 | grep Location | sed -e 's|.*/||')
                ;;
            *)
                fatal "Incorrect downloader executable '$DOWNLOADER'"
                ;;
        esac
    fi
    info "Using ${VERSION_K2S} as release"
}

# --- download from github url ---
download() {
    [ $# -eq 2 ] || fatal 'download needs exactly 2 arguments'

    case $DOWNLOADER in
        curl)
            curl -o $1 -sfL $2
            ;;
        wget)
            wget -qO $1 $2
            ;;
        *)
            fatal "Incorrect executable '$DOWNLOADER'"
            ;;
    esac

    # Abort if download command failed
    [ $? -eq 0 ] || fatal 'Download failed'
}

# --- download hash from github url ---
download_hash() {
    if [ -n "${INSTALL_K2S_COMMIT}" ]; then
        HASH_URL=${STORAGE_URL}/k2s${SUFFIX}-${INSTALL_K2S_COMMIT}.sha256sum
    else
        HASH_URL=${GITHUB_URL}/download/${VERSION_K2S}/sha256sum-${ARCH}.txt
    fi
    info "Downloading hash ${HASH_URL}"
    download ${TMP_HASH} ${HASH_URL}
    HASH_EXPECTED=$(grep " k2s${SUFFIX}$" ${TMP_HASH})
    HASH_EXPECTED=${HASH_EXPECTED%%[[:blank:]]*}
}

# --- check hash against installed version ---
installed_hash_matches() {
    if [ -x ${BIN_DIR}/k2s ]; then
        HASH_INSTALLED=$(sha256sum ${BIN_DIR}/k2s)
        HASH_INSTALLED=${HASH_INSTALLED%%[[:blank:]]*}
        if [ "${HASH_EXPECTED}" = "${HASH_INSTALLED}" ]; then
            return
        fi
    fi
    return 1
}

# --- download binary from github url ---
download_binary() {
    if [ -n "${INSTALL_K2S_COMMIT}" ]; then
        BIN_URL=${STORAGE_URL}/k2s${SUFFIX}-${INSTALL_K2S_COMMIT}
    else
        BIN_URL=${GITHUB_URL}/download/${VERSION_K2S}/k2s${SUFFIX}
    fi
    info "Downloading binary ${BIN_URL}"
    download ${TMP_BIN} ${BIN_URL}
}

# --- verify downloaded binary hash ---
verify_binary() {
    info "Verifying binary download"
    HASH_BIN=$(sha256sum ${TMP_BIN})
    HASH_BIN=${HASH_BIN%%[[:blank:]]*}
    if [ "${HASH_EXPECTED}" != "${HASH_BIN}" ]; then
        fatal "Download sha256 does not match ${HASH_EXPECTED}, got ${HASH_BIN}"
    fi
}

# --- setup permissions and move binary to system directory ---
setup_binary() {
    chmod 755 ${TMP_BIN}
    info "Installing K2S to ${BIN_DIR}/K2S"
    $SUDO chown root:root ${TMP_BIN}
    $SUDO mv -f ${TMP_BIN} ${BIN_DIR}/k2s

    if command -v getenforce >/dev/null 2>&1; then
        if [ "Disabled" != $(getenforce) ]; then
	    info 'SELinux is enabled, setting permissions'
	    if ! $SUDO semanage fcontext -l | grep "${BIN_DIR}/k2s" > /dev/null 2>&1; then
	        $SUDO semanage fcontext -a -t bin_t "${BIN_DIR}/k2s"
	    fi
	    $SUDO restorecon -v ${BIN_DIR}/k2s > /dev/null
        fi
    fi
}

# --- download and verify K2S ---
download_and_verify() {
    if can_skip_download; then
       info "Skipping K2S download and verify"
       verify_K2S_is_executable
       return
    fi

    setup_verify_arch
    verify_downloader curl || verify_downloader wget || fatal 'Can not find curl or wget for downloading files'
    verify_semanage
    setup_tmp
    get_release_version
    download_hash

    if installed_hash_matches; then
        info "Skipping binary downloaded, installed K2S matches hash"
        return
    fi

    download_binary
    verify_binary
    setup_binary
}

# --- add additional utility links ---
create_symlinks() {
    [ "${INSTALL_K2S_BIN_DIR_READ_ONLY}" = "true" ] && return
    [ "${INSTALL_K2S_SYMLINK}" = "skip" ] && return

    for cmd in kubectl crictl ctr; do
        if [ ! -e ${BIN_DIR}/${cmd} ] || [ "${INSTALL_K2S_SYMLINK}" = force ]; then
            which_cmd=$(which ${cmd} 2>/dev/null || true)
            if [ -z "${which_cmd}" ] || [ "${INSTALL_K2S_SYMLINK}" = force ]; then
                info "Creating ${BIN_DIR}/${cmd} symlink to k2s"
                $SUDO ln -sf k2s ${BIN_DIR}/${cmd}
            else
                info "Skipping ${BIN_DIR}/${cmd} symlink to k2s, command exists in PATH at ${which_cmd}"
            fi
        else
            info "Skipping ${BIN_DIR}/${cmd} symlink to k2s, already exists"
        fi
    done
}

# --- create killall script ---
create_killall() {
    [ "${INSTALL_K2S_BIN_DIR_READ_ONLY}" = true ] && return
    info "Creating killall script ${KILLALL_K2S_SH}"
    $SUDO tee ${KILLALL_K2S_SH} >/dev/null << \EOF
#!/bin/sh
[ $(id -u) -eq 0 ] || exec sudo $0 $@

for bin in /var/lib/rancher/k2s/data/**/bin/; do
    [ -d $bin ] && export PATH=$PATH:$bin:$bin/aux
done

set -x

for service in /etc/systemd/system/k2s*.service; do
    [ -s $service ] && systemctl stop $(basename $service)
done

for service in /etc/init.d/k2s*; do
    [ -x $service ] && $service stop
done

pschildren() {
    ps -e -o ppid= -o pid= | \
    sed -e 's/^\s*//g; s/\s\s*/\t/g;' | \
    grep -w "^$1" | \
    cut -f2
}

pstree() {
    for pid in $@; do
        echo $pid
        for child in $(pschildren $pid); do
            pstree $child
        done
    done
}

killtree() {
    kill -9 $(
        { set +x; } 2>/dev/null;
        pstree $@;
        set -x;
    ) 2>/dev/null
}

getshims() {
    ps -e -o pid= -o args= | sed -e 's/^ *//; s/\s\s*/\t/;' | grep -w 'k2s/data/[^/]*/bin/containerd-shim' | cut -f1
}

killtree $({ set +x; } 2>/dev/null; getshims; set -x)

do_unmount() {
    { set +x; } 2>/dev/null
    MOUNTS=
    while read ignore mount ignore; do
        MOUNTS="$mount\n$MOUNTS"
    done </proc/self/mounts
    MOUNTS=$(printf $MOUNTS | grep "^$1" | sort -r)
    if [ -n "${MOUNTS}" ]; then
        set -x
        umount ${MOUNTS}
    else
        set -x
    fi
}

do_unmount '/run/k2s'
do_unmount '/var/lib/Yuwenfeng2019/K2S'
do_unmount '/var/lib/kubelet/pods'
do_unmount '/run/netns/cni-'

# Delete network interface(s) that match 'master cni0'
ip link show 2>/dev/null | grep 'master cni0' | while read ignore iface ignore; do
    iface=${iface%%@*}
    [ -z "$iface" ] || ip link delete $iface
done
ip link delete cni0
ip link delete flannel.1
rm -rf /var/lib/cni/
iptables-save | grep -v KUBE- | grep -v CNI- | iptables-restore
EOF
    $SUDO chmod 755 ${KILLALL_K2S_SH}
    $SUDO chown root:root ${KILLALL_K2S_SH}
}

# --- create uninstall script ---
create_uninstall() {
    [ "${INSTALL_K2S_BIN_DIR_READ_ONLY}" = true ] && return
    info "Creating uninstall script ${UNINSTALL_K2S_SH}"
    $SUDO tee ${UNINSTALL_K2S_SH} >/dev/null << EOF
#!/bin/sh
set -x
[ \`id -u\` -eq 0 ] || exec sudo \$0 \$@

${KILLALL_K2S_SH}

if which systemctl; then
    systemctl disable ${SYSTEM_NAME}
    systemctl reset-failed ${SYSTEM_NAME}
    systemctl daemon-reload
fi
if which rc-update; then
    rc-update delete ${SYSTEM_NAME} default
fi

rm -f ${FILE_K2S_SERVICE}
rm -f ${FILE_K2S_ENV}

remove_uninstall() {
    rm -f ${UNINSTALL_K2S_SH}
}
trap remove_uninstall EXIT

if (ls ${SYSTEMD_DIR}/k2s*.service || ls /etc/init.d/k2s*) >/dev/null 2>&1; then
    set +x; echo "Additional k2s services installed, skipping uninstall of k2s"; set -x
    exit
fi

for cmd in kubectl crictl ctr; do
    if [ -L ${BIN_DIR}/\$cmd ]; then
        rm -f ${BIN_DIR}/\$cmd
    fi
done

rm -rf /etc/Yuwenfeng2019/K2S
rm -rf /var/lib/Yuwenfeng2019/K2S
rm -rf /var/lib/kubelet
rm -f ${BIN_DIR}/k2s
rm -f ${KILLALL_K2S_SH}
EOF
    $SUDO chmod 755 ${UNINSTALL_K2S_SH}
    $SUDO chown root:root ${UNINSTALL_K2S_SH}
}

# --- disable current service if loaded --
systemd_disable() {
    $SUDO rm -f /etc/systemd/system/${SERVICE_K2S} || true
    $SUDO rm -f /etc/systemd/system/${SERVICE_K2S}.env || true
    $SUDO systemctl disable ${SYSTEM_NAME} >/dev/null 2>&1 || true
}

# --- capture current env and create file containing K2S_ variables ---
create_env_file() {
    info "env: Creating environment file ${FILE_K2S_ENV}"
    UMASK=$(umask)
    umask 0377
    env | grep '^K2S_' | $SUDO tee ${FILE_K2S_ENV} >/dev/null
    env | egrep -i '^(NO|HTTP|HTTPS)_PROXY' | $SUDO tee -a ${FILE_K2S_ENV} >/dev/null
    umask $UMASK
}

# --- write systemd service file ---
create_systemd_service_file() {
    info "systemd: Creating service file ${FILE_K2S_SERVICE}"
    $SUDO tee ${FILE_K2S_SERVICE} >/dev/null << EOF
[Unit]
Description=Lightweight Kubernetes
Documentation=https://k2s.io
Wants=network-online.target

[Install]
WantedBy=multi-user.target

[Service]
Type=${SYSTEMD_TYPE}
EnvironmentFile=${FILE_K2S_ENV}
KillMode=process
Delegate=yes
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
Restart=always
RestartSec=5s
ExecStartPre=-/sbin/modprobe br_netfilter
ExecStartPre=-/sbin/modprobe overlay
ExecStart=${BIN_DIR}/k2s \\
    ${CMD_K2S_EXEC}

EOF
}

# --- write openrc service file ---
create_openrc_service_file() {
    LOG_FILE=/var/log/${SYSTEM_NAME}.log

    info "openrc: Creating service file ${FILE_K2S_SERVICE}"
    $SUDO tee ${FILE_K2S_SERVICE} >/dev/null << EOF
#!/sbin/openrc-run

depend() {
    after network-online
    want cgroups
}

start_pre() {
    rm -f /tmp/k2s.*
}

supervisor=supervise-daemon
name="${SYSTEM_NAME}"
command="${BIN_DIR}/k2s"
command_args="$(escape_dq "${CMD_K2S_EXEC}")
    >>${LOG_FILE} 2>&1"

output_log=${LOG_FILE}
error_log=${LOG_FILE}

pidfile="/var/run/${SYSTEM_NAME}.pid"
respawn_delay=5

set -o allexport
if [ -f /etc/environment ]; then source /etc/environment; fi
if [ -f ${FILE_K2S_ENV} ]; then source ${FILE_K2S_ENV}; fi
set +o allexport
EOF
    $SUDO chmod 0755 ${FILE_K2S_SERVICE}

    $SUDO tee /etc/logrotate.d/${SYSTEM_NAME} >/dev/null << EOF
${LOG_FILE} {
	missingok
	notifempty
	copytruncate
}
EOF
}

# --- write systemd or openrc service file ---
create_service_file() {
    [ "${HAS_SYSTEMD}" = true ] && create_systemd_service_file
    [ "${HAS_OPENRC}" = true ] && create_openrc_service_file
    return 0
}

# --- get hashes of the current k2s bin and service files
get_installed_hashes() {
    $SUDO sha256sum ${BIN_DIR}/k2s ${FILE_K2S_SERVICE} ${FILE_K2S_ENV} 2>&1 || true
}

# --- enable and start systemd service ---
systemd_enable() {
    info "systemd: Enabling ${SYSTEM_NAME} unit"
    $SUDO systemctl enable ${FILE_K2S_SERVICE} >/dev/null
    $SUDO systemctl daemon-reload >/dev/null
}

systemd_start() {
    info "systemd: Starting ${SYSTEM_NAME}"
    $SUDO systemctl restart ${SYSTEM_NAME}
}

# --- enable and start openrc service ---
openrc_enable() {
    info "openrc: Enabling ${SYSTEM_NAME} service for default runlevel"
    $SUDO rc-update add ${SYSTEM_NAME} default >/dev/null
}

openrc_start() {
    info "openrc: Starting ${SYSTEM_NAME}"
    $SUDO ${FILE_K2S_SERVICE} restart
}

# --- startup systemd or openrc service ---
service_enable_and_start() {
    [ "${INSTALL_K2S_SKIP_ENABLE}" = true ] && return

    [ "${HAS_SYSTEMD}" = true ] && systemd_enable
    [ "${HAS_OPENRC}" = true ] && openrc_enable

    [ "${INSTALL_K2S_SKIP_START}" = "true" ] && return

    POST_INSTALL_HASHES=$(get_installed_hashes)
    if [ "${PRE_INSTALL_HASHES}" = "${POST_INSTALL_HASHES}" ]; then
        info 'No change detected so skipping service start'
        return
    fi

    [ "${HAS_SYSTEMD}" = true ] && systemd_start
    [ "${HAS_OPENRC}" = true ] && openrc_start
    return 0
}

# --- re-evaluate args to include env command ---
eval set -- $(escape "${INSTALL_K2S_EXEC}") $(quote "$@")

# --- run the install process --
{
    verify_system
    setup_env "$@"
    download_and_verify
    create_symlinks
    create_killall
    create_uninstall
    systemd_disable
    create_env_file
    create_service_file
    service_enable_and_start
}
