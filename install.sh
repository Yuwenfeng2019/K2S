#!/bin/sh
set -e

# Usage:
#   curl ... | ENV_VAR=... sh -
#       or
#   ENV_VAR=... ./install.sh
#
# Example:
#   Installing a server without an agent:
#     curl ... | INSTALL_K2S_EXEC="--disable-agent" sh -
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
#   - INSTALL_K2S_SKIP_START
#     If set to true will not start k2s service.
#
#   - INSTALL_K2S_VERSION
#     Version of k2s to download from github. Will attempt to download the
#     latest version if not specified.
#
#   - INSTALL_K2S_BIN_DIR
#     Directory to install K2S binary, links, and uninstall script to, or use
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
#       curl ... | INSTALL_K2S_EXEC="--disable-agent" sh -s -
#       curl ... | INSTALL_K2S_EXEC="server --disable-agent" sh -s -
#       curl ... | INSTALL_K2S_EXEC="server" sh -s - --disable-agent
#       curl ... | sh -s - server --disable-agent
#       curl ... | sh -s - --disable-agent
#
#   - INSTALL_K2S_NAME
#     Name of systemd service to create, will default from the K2S exec command
#     if not specified. If specified the name will be prefixed with 'K2S-'.
#
#   - INSTALL_K2S_TYPE
#     Type of systemd service to create, will default from the K2S exec command
#     if not specified.

GITHUB_URL=https://github.com/Yuwenfeng2019/K2S/releases

# --- helper functions for logs ---
info()
{
    echo "[INFO] " "$@"
}
fatal()
{
    echo "[ERROR] " "$@"
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
        printf "%s\n" "$arg" | sed "s/'/'\\\\''/g;1s/^/'/;\$s/\$/'/"
    done
}

# --- add indentation and trailing slash to quoted args ---
quote_indent() {
    printf ' \\'"\n"
    for arg in "$@"; do
        printf "\t%s "'\\'"\n" "$(quote "$arg")"
    done
}

# --- escape most punctuation characters, except quotes, forward slash, and space ---
escape() {
    printf "%s" "$@" | sed -e 's/\([][!#$%&()*;<=>?\_`{|}]\)/\\\1/g;'
}

# --- escape double quotes ---
escape_dq() {
    printf "%s" "$@" | sed -e 's/"/\\"/g'
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
            CMD_K2S="$1"
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
    valid_chars=$(printf "%s" "${SYSTEM_NAME}" | sed -e 's/[][!#$%&()*;<=>?\_`{|}/[:space:]]/^/g;' )
    if [ "${SYSTEM_NAME}" != "${valid_chars}"  ]; then
        invalid_chars=$(printf "%s" "${valid_chars}" | sed -e 's/[^^]/ /g')
        fatal "Invalid characters for system name:
            ${SYSTEM_NAME}
            ${invalid_chars}"
    fi

    # --- set related files from system name ---
    SERVICE_K2S=${SYSTEM_NAME}.service
    UNINSTALL_K2S_SH=${SYSTEM_NAME}-uninstall.sh
    KILLALL_K2S_SH=k2s-killall.sh

    # --- use sudo if we are not already root ---
    SUDO=sudo
    if [ `id -u` = 0 ]; then
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
        BIN_DIR="/usr/local/bin"
    fi

    # --- use systemd directory if defined or create default ---
    if [ -n "${INSTALL_K2S_SYSTEMD_DIR}" ]; then
        SYSTEMD_DIR="${INSTALL_K2S_SYSTEMD_DIR}"
    else
        SYSTEMD_DIR="/etc/systemd/system"
    fi

    # --- use service or environment location depending on systemd/openrc ---
    if [ "${HAS_SYSTEMD}" = "true" ]; then
        FILE_K2S_SERVICE=${SYSTEMD_DIR}/${SERVICE_K2S}
        FILE_K2S_ENV=${SYSTEMD_DIR}/${SERVICE_K2S}.env
    elif [ "${HAS_OPENRC}" = "true" ]; then
        $SUDO mkdir -p /etc/rancher/k2s
        FILE_K2S_SERVICE=/etc/init.d/${SYSTEM_NAME}
        FILE_K2S_ENV=/etc/rancher/k2s/${SYSTEM_NAME}.env
    fi

    # --- get hash of config & exec for currently installed k2s ---
    PRE_INSTALL_HASHES=`get_installed_hashes`

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
        ARCH=`uname -m`
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

# --- fatal if no curl ---
verify_curl() {
    if [ -z `which curl || true` ]; then
        fatal "Can not find curl for downloading files"
    fi
}

# --- create tempory directory and cleanup when done ---
setup_tmp() {
    TMP_DIR=`mktemp -d -t K2S-install.XXXXXXXXXX`
    TMP_HASH=${TMP_DIR}/K2S.hash
    TMP_BIN=${TMP_DIR}/K2S.bin
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
    if [ -n "${INSTALL_K2S_VERSION}" ]; then
        VERSION_K2S="${INSTALL_K2S_VERSION}"
    else
        info "Finding latest release"
        VERSION_K2S=`curl -w "%{url_effective}" -I -L -s -S ${GITHUB_URL}/latest -o /dev/null | sed -e 's|.*/||'`
    fi
    info "Using ${VERSION_K2S} as release"
}

# --- download hash from github url ---
download_hash() {
    HASH_URL=${GITHUB_URL}/download/${VERSION_K2S}/sha256sum-${ARCH}.txt
    info "Downloading hash ${HASH_URL}"
    curl -o ${TMP_HASH} -sfL ${HASH_URL} || fatal "Hash download failed"
    HASH_EXPECTED=`grep " k2s${SUFFIX}$" ${TMP_HASH} | awk '{print $1}'`
}

# --- check hash against installed version ---
installed_hash_matches() {
    if [ -x ${BIN_DIR}/K2S ]; then
        HASH_INSTALLED=`sha256sum ${BIN_DIR}/K2S | awk '{print $1}'`
        if [ "${HASH_EXPECTED}" = "${HASH_INSTALLED}" ]; then
            return
        fi
    fi
    return 1
}

# --- download binary from github url ---
download_binary() {
    BIN_URL=${GITHUB_URL}/download/${VERSION_K2S}/K2S${SUFFIX}
    info "Downloading binary ${BIN_URL}"
    curl -o ${TMP_BIN} -sfL ${BIN_URL} || fatal "Binary download failed"
}

# --- verify downloaded binary hash ---
verify_binary() {
    info "Verifying binary download"
    HASH_BIN=`sha256sum ${TMP_BIN} | awk '{print $1}'`
    if [ "${HASH_EXPECTED}" != "${HASH_BIN}" ]; then
        fatal "Download sha256 does not match ${HASH_EXPECTED}, got ${HASH_BIN}"
    fi
}

# --- setup permissions and move binary to system directory ---
setup_binary() {
    chmod 755 ${TMP_BIN}
    info "Installing K2S to ${BIN_DIR}/K2S"
    $SUDO chown root:root ${TMP_BIN}
    $SUDO mv -f ${TMP_BIN} ${BIN_DIR}/K2S
    if command -v getenforce > /dev/null 2>&1; then
        if [ "Disabled" != `getenforce` ]; then
            info "SeLinux is enabled, setting permissions"
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
    verify_curl
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
    if [ ! -e ${BIN_DIR}/kubectl ]; then
        info "Creating ${BIN_DIR}/kubectl symlink to K2S"
        $SUDO ln -s K2S ${BIN_DIR}/kubectl
    fi

    if [ ! -e ${BIN_DIR}/crictl ]; then
        info "Creating ${BIN_DIR}/crictl symlink to K2S"
        $SUDO ln -s K2S ${BIN_DIR}/crictl
    fi
}


# --- create killall script ---
create_killall() {
    [ "${INSTALL_K2S_BIN_DIR_READ_ONLY}" = "true" ] && return
    info "Creating killall script ${BIN_DIR}/${KILLALL_K2S_SH}"
    $SUDO tee ${BIN_DIR}/${KILLALL_K2S_SH} >/dev/null << \EOF
#!/bin/sh
set -x
[ `id -u` = 0 ] || exec sudo $0 $@

for bin in /var/lib/yuwenfeng2019/k2s/data/**/bin/; do
    [ -d $bin ] && export PATH=$bin:$PATH
done

for service in /etc/systemd/system/k2s*.service; do
    [ -s $service ] && systemctl stop $(basename $service)
done

for service in /etc/init.d/k2s*; do
    [ -x $service ] && $service stop
done

pstree() {
    for pid in $@; do
        echo $pid
        pstree $(ps -o ppid= -o pid= | awk "\$1==$pid {print \$2}")
    done
}

killtree() {
    [ $# -ne 0 ] && kill $(set +x; pstree $@; set -x)
}

killtree $(lsof | sed -e 's/^[^0-9]*//g; s/  */\t/g' | grep -w 'k2s/data/[^/]*/bin/containerd-shim' | cut -f1 | sort -n -u)

do_unmount() {
    MOUNTS=`cat /proc/self/mounts | awk '{print $2}' | grep "^$1" | sort -r`
    if [ -n "${MOUNTS}" ]; then
        umount ${MOUNTS}
    fi
}

do_unmount '/run/k2s'
do_unmount '/var/lib/rancher/k2s'

nets=$(ip link show | grep 'master cni0' | awk -F': ' '{print $2}' | sed -e 's|@.*||')
for iface in $nets; do
    ip link delete $iface;
done
ip link delete cni0
ip link delete flannel.1
rm -rf /var/lib/cni/
EOF
    $SUDO chmod 755 ${BIN_DIR}/${KILLALL_K2S_SH}
    $SUDO chown root:root ${BIN_DIR}/${KILLALL_K2S_SH}
}

# --- create uninstall script ---
create_uninstall() {
    [ "${INSTALL_K2S_BIN_DIR_READ_ONLY}" = "true" ] && return
    info "Creating uninstall script ${BIN_DIR}/${UNINSTALL_K2S_SH}"
    $SUDO tee ${BIN_DIR}/${UNINSTALL_K2S_SH} >/dev/null << EOF
#!/bin/sh
set -x
[ \`id -u\` = 0 ] || exec sudo \$0 \$@

${BIN_DIR}/${KILLALL_K2S_SH}

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
    rm -f ${BIN_DIR}/${UNINSTALL_K2S_SH}
}
trap remove_uninstall EXIT

if (ls ${SYSTEMD_DIR}/k2s*.service || ls /etc/init.d/k2s*) >/dev/null 2>&1; then
    set +x; echo "Additional k2s services installed, skipping uninstall of k2s"; set -x
    exit
fi

if [ -L ${BIN_DIR}/kubectl ]; then
    rm -f ${BIN_DIR}/kubectl
fi
if [ -L ${BIN_DIR}/crictl ]; then
    rm -f ${BIN_DIR}/crictl
fi

rm -rf /etc/yuwenfeng2019/k2s
rm -rf /var/lib/yuwenfeng2019/k2s
rm -f ${BIN_DIR}/k2s
rm -f ${BIN_DIR}/${KILLALL_K2S_SH}
EOF
    $SUDO chmod 755 ${BIN_DIR}/${UNINSTALL_K2S_SH}
    $SUDO chown root:root ${BIN_DIR}/${UNINSTALL_K2S_SH}
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
    UMASK=`umask`
    umask 0377
    env | grep '^K2S_' | $SUDO tee ${FILE_K2S_ENV} >/dev/null
    umask $UMASK
}

# --- write systemd service file ---
create_systemd_service_file() {
    info "systemd: Creating service file ${FILE_K2S_SERVICE}"
    $SUDO tee ${FILE_K2S_SERVICE} >/dev/null << EOF
[Unit]
Description=Lightweight Kubernetes
Documentation=https://k2s.io
After=network-online.target

[Service]
Type=${SYSTEMD_TYPE}
EnvironmentFile=${FILE_K2S_ENV}
ExecStartPre=-/sbin/modprobe br_netfilter
ExecStartPre=-/sbin/modprobe overlay
ExecStart=${BIN_DIR}/k2s \\
    ${CMD_K2S_EXEC}

KillMode=process
Delegate=yes
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
Restart=always

[Install]
WantedBy=multi-user.target
EOF
}

# --- write openrc service file ---
create_openrc_service_file() {
    LOG_FILE=/var/log/${SYSTEM_NAME}.log

    info "openrc: Creating service file ${FILE_K2S_SERVICE}"
    $SUDO tee ${FILE_K2S_SERVICE} >/dev/null << EOF
#!/sbin/openrc-run

depend() {
    after net-online
    need net
}

start_pre() {
    rm -f /tmp/k2s.*
}

supervisor=supervise-daemon
name="${SYSTEM_NAME}"
command="${BIN_DIR}/k2s"
command_args="$(escape_dq "${CMD_K2S_EXEC}")
    >>${LOG_FILE} 2>&1"

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
    [ "${HAS_SYSTEMD}" = "true" ] && create_systemd_service_file
    [ "${HAS_OPENRC}" = "true" ] && create_openrc_service_file
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
    [ "${HAS_SYSTEMD}" = "true" ] && systemd_enable
    [ "${HAS_OPENRC}" = "true" ] && openrc_enable

    [ "${INSTALL_K2S_SKIP_START}" = "true" ] && return

    POST_INSTALL_HASHES=`get_installed_hashes`
    if [ "${PRE_INSTALL_HASHES}" = "${POST_INSTALL_HASHES}" ]; then
        info "No change detected so skipping service start"
        return
    fi

    [ "${HAS_SYSTEMD}" = "true" ] && systemd_start
    [ "${HAS_OPENRC}" = "true" ] && openrc_start
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
