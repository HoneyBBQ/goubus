#!/usr/bin/env bash
# Start or stop an ephemeral OpenWrt container (21.02–25.12) exposing ubus over
# Unix domain socket, HTTP JSON-RPC, and HTTPS JSON-RPC for live integration tests.
#
#   ./scripts/openwrt-lab.sh start <21-02|22-03|23-05|24-10|25-12> <work_dir>
#   ./scripts/openwrt-lab.sh stop <work_dir>
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$here/.." && pwd)"

usage() {
  echo "usage: openwrt-lab.sh start <21-02|22-03|23-05|24-10|25-12> <work_dir> | stop <work_dir>" >&2
  exit 1
}

stop_resources() {
  local work_dir="$1"
  if [ -f "$work_dir/bridge.pid" ]; then
    local pid
    pid="$(cat "$work_dir/bridge.pid" 2>/dev/null || true)"
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      for _ in 1 2 3 4 5; do
        kill -0 "$pid" 2>/dev/null || break
        sleep 1
      done
      kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$work_dir/bridge.pid" "$work_dir/bridge.ready" "$work_dir/ubus.sock"
  fi

  if [ -f "$work_dir/container.name" ]; then
    local cname
    cname="$(cat "$work_dir/container.name" 2>/dev/null || true)"
    if [ -n "$cname" ]; then
      docker rm -f "$cname" >/dev/null 2>&1 || true
    fi
    rm -f "$work_dir/container.name"
  fi
}

cmd_stop() {
  local raw_dir="${1:-}"
  [ -n "$raw_dir" ] || usage
  if [ ! -d "$raw_dir" ]; then
    return 0
  fi
  local work_dir
  work_dir="$(cd "$raw_dir" && pwd)"
  stop_resources "$work_dir"
  if [ -n "$work_dir" ] && [ "$work_dir" != "/" ]; then
    rm -rf "$work_dir"
  fi
}

resolve_release() {
  case "$1" in
    21-02|openwrt-21-02|21.02)
      IMAGE_KEY="openwrt-21-02"
      RELEASE_VER="21.02"
      ;;
    22-03|openwrt-22-03|22.03)
      IMAGE_KEY="openwrt-22-03"
      RELEASE_VER="22.03"
      ;;
    23-05|openwrt-23-05|23.05)
      IMAGE_KEY="openwrt-23-05"
      RELEASE_VER="23.05"
      ;;
    24-10|openwrt-24-10|24.10)
      IMAGE_KEY="openwrt-24-10"
      RELEASE_VER="24.10"
      ;;
    25-12|openwrt-25-12|25.12)
      IMAGE_KEY="openwrt-25-12"
      RELEASE_VER="25.12"
      ;;
    *)
      echo "unsupported OpenWrt target: $1 (expected 21-02, 22-03, 23-05, 24-10, or 25-12)" >&2
      exit 1
      ;;
  esac
}

write_lab_files() {
  local lab_dir="$1"
  mkdir -p "$lab_dir"

  cat > "$lab_dir/superuser.json" <<'JSON'
{
  "superuser": {
    "description": "Super user access role for live integration tests",
    "read": {
      "ubus": { "*": [ "*" ] },
      "uci": [ "*" ],
      "file": { "*": [ "read", "write", "exec", "stat" ] }
    },
    "write": {
      "ubus": { "*": [ "*" ] },
      "uci": [ "*" ],
      "file": { "*": [ "read", "write", "exec", "stat" ] }
    }
  }
}
JSON

  cat > "$lab_dir/init.sh" <<'SH'
#!/bin/sh
set -eu

mkdir -p /tmp/run/ubus /tmp/lock /www /usr/share/rpcd/acl.d
chmod 0777 /tmp/run/ubus
if [ ! -f /etc/config/system ] && [ -x /bin/config_generate ]; then
  /bin/config_generate >/dev/null 2>&1
fi
cp /lab/superuser.json /usr/share/rpcd/acl.d/superuser.json
printf '%s\n%s\n' "$UBUS_LAB_PASSWORD" "$UBUS_LAB_PASSWORD" | /bin/passwd root >/dev/null 2>&1

/sbin/ubusd -s /tmp/run/ubus/ubus.sock &
i=0
while [ ! -S /tmp/run/ubus/ubus.sock ]; do
  i=$((i + 1))
  if [ "$i" -ge 30 ]; then
    echo "ubusd socket did not appear" >&2
    exit 1
  fi
  sleep 1
done
chmod 0666 /tmp/run/ubus/ubus.sock

/lab/testlab-linux bridge \
  -listen-network tcp \
  -listen-addr 0.0.0.0:12000 \
  -target-network unix \
  -target-addr /tmp/run/ubus/ubus.sock &

/sbin/rpcd -s /tmp/run/ubus/ubus.sock &
i=0
while ! ubus -s /tmp/run/ubus/ubus.sock list session >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -ge 30 ]; then
    echo "rpcd session object did not appear" >&2
    exit 1
  fi
  sleep 1
done

exec /usr/sbin/uhttpd -f \
  -p 0.0.0.0:80 \
  -s 0.0.0.0:443 \
  -C /tls/server.pem \
  -K /tls/server.key \
  -u /ubus \
  -U /tmp/run/ubus/ubus.sock \
  -h /www
SH

  chmod 0644 "$lab_dir/superuser.json"
  chmod 0755 "$lab_dir/init.sh"
}

cmd_start() {
  local target="${1:-}"
  local raw_dir="${2:-}"
  [ -n "$target" ] && [ -n "$raw_dir" ] || usage

  resolve_release "$target"
  mkdir -p "$raw_dir"
  local work_dir
  work_dir="$(cd "$raw_dir" && pwd)"
  stop_resources "$work_dir"

  local image_ref="${OPENWRT_LAB_IMAGE:-$("$here/ci-image.sh" "$IMAGE_KEY")}"
  mkdir -p "$work_dir/bin" "$work_dir/tls" "$work_dir/lab"

  (cd "$repo_root" && go build -o "$work_dir/bin/testlab-host" ./internal/tools/testlab)
  (cd "$repo_root" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o "$work_dir/lab/testlab-linux" ./internal/tools/testlab)
  chmod 0755 "$work_dir/lab/testlab-linux"

  local password
  password="$("$work_dir/bin/testlab-host" rand)"
  "$work_dir/bin/testlab-host" tls -dir "$work_dir/tls"
  chmod 0644 "$work_dir/tls/ca.pem" "$work_dir/tls/server.pem" "$work_dir/tls/server.key"

  write_lab_files "$work_dir/lab"

  local suffix
  suffix="$("$work_dir/bin/testlab-host" rand | cut -c1-8)"
  local container_name="goubus-${IMAGE_KEY}-${suffix}"
  echo "$container_name" > "$work_dir/container.name"

  # Bridge the container ubusd socket through testlab-linux (running as uid=0
  # inside the container) on every host OS so ubusd's SO_PEERCRED check sees
  # root regardless of the host user UID (for example uid=1001 on Linux CI).
  local socket_path="$work_dir/ubus.sock"
  docker run -d --platform linux/amd64 \
    --name "$container_name" \
    -e UBUS_LAB_PASSWORD="$password" \
    -p 127.0.0.1::80 \
    -p 127.0.0.1::443 \
    -p 127.0.0.1::12000 \
    -v "$work_dir/tls:/tls:ro" \
    -v "$work_dir/lab:/lab:ro" \
    "$image_ref" /bin/sh /lab/init.sh >/dev/null

  local bridge_addr
  bridge_addr="$(docker port "$container_name" 12000/tcp | head -n 1)"
  rm -f "$work_dir/bridge.ready"
  nohup "$work_dir/bin/testlab-host" bridge \
    -listen-network unix \
    -listen-addr "$socket_path" \
    -target-network tcp \
    -target-addr "$bridge_addr" \
    -ready-file "$work_dir/bridge.ready" \
    >"$work_dir/bridge.log" 2>&1 </dev/null &
  echo "$!" > "$work_dir/bridge.pid"

  for _ in 1 2 3 4 5 6 7 8 9 10; do
    [ -f "$work_dir/bridge.ready" ] && break
    sleep 1
  done

  local http_addr https_addr http_url https_url
  http_addr="$(docker port "$container_name" 80/tcp | head -n 1)"
  https_addr="$(docker port "$container_name" 443/tcp | head -n 1)"
  http_url="http://$http_addr"
  https_url="https://$https_addr"

  if ! "$work_dir/bin/testlab-host" wait \
    -socket "$socket_path" \
    -http "$http_url" \
    -https "$https_url" \
    -ca "$work_dir/tls/ca.pem" \
    -user root \
    -password "$password" \
    -timeout 45s; then
    docker logs "$container_name" >&2 || true
    if [ -f "$work_dir/bridge.log" ]; then
      cat "$work_dir/bridge.log" >&2 || true
    fi
    stop_resources "$work_dir"
    exit 1
  fi

  cat > "$work_dir/env.sh" <<EOF
export UBUS_TEST_SOCKET='$socket_path'
export UBUS_TEST_HTTP_URL='$http_url'
export UBUS_TEST_HTTPS_URL='$https_url'
export UBUS_TEST_CA_FILE='$work_dir/tls/ca.pem'
export UBUS_TEST_USER='root'
export UBUS_TEST_PASSWORD='$password'
export UBUS_TEST_RELEASE='$RELEASE_VER'
EOF

  echo "$work_dir/env.sh"
}

action="${1:-}"
[ -n "$action" ] || usage
shift

case "$action" in
  start) cmd_start "$@" ;;
  stop) cmd_stop "$@" ;;
  *) usage ;;
esac
