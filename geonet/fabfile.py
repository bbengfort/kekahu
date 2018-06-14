# fabfile
# Fabric commands for managing KeKahu on a cluster
#
# Author:  Benjamin Bengfort <benjamin@bengfort.com>
# Created: Thu Jun 14 13:28:58 2018 -0400
#
# ID: fabfile.py [] benjamin@bengfort.com $

"""
Fabric commands for managing KeKahu on a cluster
"""

##########################################################################
## Imports
##########################################################################

import os
import json

from StringIO import StringIO
from tabulate import tabulate
from collections import Counter
from dotenv import load_dotenv, find_dotenv

from fabric.api import env, run, cd, get, put, hide, sudo
from fabric.api import parallel, task, runs_once, execute


##########################################################################
## Environment Helpers
##########################################################################

def load_json(path):
    with open(path, 'r') as f:
        return json.load(f)


##########################################################################
## Environment
##########################################################################

## Load the environment
load_dotenv(find_dotenv())

## Local paths
FIXTURES = os.path.join(os.path.dirname(__file__), "data")
HOSTINFO = os.path.join(FIXTURES, "hosts.json")
TOKENS   = os.path.join(FIXTURES, "tokens.json")
SERVICE  = os.path.join(FIXTURES, "kekahu.service")

## Remote paths
KEKAHU_REPO = "~/workspace/go/src/github.com/bbengfort/kekahu"
KEKAHU_CONFIG = "/etc/kekahu.json"
KEKAHU_BIN = "/usr/local/bin/kekahu"
KEKAHU_SYSTEMD = "/etc/systemd/system/kekahu.service"

## Load hosts from the hosts file
hosts = load_json(HOSTINFO)
addrs = {info['hostname']: host for host, info in hosts.items()}

## Fabric Env
env.user = "ubuntu"
env.hosts = sorted(list(hosts.keys()))
env.colorize_errors = True
env.use_ssh_config = True
env.forward_agent = True


##########################################################################
## Fabric Commands
##########################################################################

@task
@parallel
def update():
    """
    Update the kekahu by pulling from GitHub and installing with go install.
    """
    with cd(KEKAHU_REPO):
        run("git pull")
        run("make deps")
        run("go install ./...")


@parallel
def fetch_version():
    """
    Get the current Kekahu version from the host
    """
    return run("{} -version".format(KEKAHU_BIN))


@task
@runs_once
def version():

    with hide("output", "running"):
        data = execute(fetch_version)

    n_hosts = float(len(data))
    versions = Counter(data.values())

    table = [["Version", "Replicas", "Percent"]]
    for version, count in versions.most_common():
        table.append([version, count, float(count)/ n_hosts * 100])

    print(tabulate(table, tablefmt="simple", headers="firstrow"))


@task
@parallel
def update_config():
    name = addrs[env.host]
    with open(TOKENS, "r") as f:
        tokens = json.load(f)

    for token in tokens:
        if token["name"] == name:
            api_key = token["api_key"]
            break
    else:
        raise ValueError("no token found for {}".format(name))

    config = {
      "interval": "2m",
      "jitter": "30s",
      "api_key": api_key,
      "url": "https://kahu.bengfort.com",
      "verbosity": 3,
      "peers_path": "/data/peers.json",
      "api_timeout": "5s",
      "ping_timeout": "10s"
    }

    config = StringIO(json.dumps(config, indent=2))
    put(config, KEKAHU_CONFIG, use_sudo=True)


@task
@parallel
def enable_systemd():
    put(SERVICE, KEKAHU_SYSTEMD, use_sudo=True)
    sudo("systemctl enable kekahu")
    sudo("systemctl daemon-reload")
    sudo("systemctl start kekahu")


@task
@parallel
def restart_systemd():
    sudo("systemctl daemon-reload")
    sudo("systemctl restart kekahu")
