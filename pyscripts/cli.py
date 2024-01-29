#!/usr/bin/env python
# encoding: utf-8

import api
import os
import sys
import json
import time
import argparse

SERVERS = {
    # '<host>': ('controller ip', 'host ip'),
}
username = ''
password = ''
CREDS = {
}
STORAGE_FILE = os.path.join(os.path.dirname(__file__), '.token.json')


def save_token():
    with open(STORAGE_FILE,'w') as fp:
        json.dump({
            'CREDS': CREDS,
            'time': time.time(),
        }, fp)


def load_token():
    global CREDS
    try:
        with open(STORAGE_FILE, 'r') as fp:
            data = json.load(fp)
        now = time.time()
        if (now - data['time']) < 1200:
            # 20分钟有效
            CREDS = data['CREDS']
    except Exception as e:
        CREDS = {}

def parse_args(args):
    parser = argparse.ArgumentParser(description="")
    parser.add_argument('-o', '--operator', required=True, help='执行的操作： status | poweron | poweroff')
    parser.add_argument('-H', '--host', required=True, help='对应的机器名')
    parser.add_argument('-a', '--arg', required=False, help='额外参数')
    return parser.parse_args(args)


def login(host, username, password):
    api.x_auth_token = 'yes'
    if host in CREDS:
        api.creds = CREDS[host]
        return True
    else:
        creds = api.set_iDRAC_script_session(SERVERS[host][0], username, password, 'false', 'y')
        if creds:
            CREDS[host] = creds
            return True


def get_power_state(host):
    if host not in CREDS:
        return False
    api.creds = CREDS[host]
    return api.get_current_server_power_state()


def set_power_state(host, state):
    if host not in CREDS:
        return False
    api.creds = CREDS[host]
    return api.set_server_power_state(state)


def get_status(host, *args):
    ret = get_power_state(host)
    print('Status:', ret)


def poweron(host, *args):
    if get_power_state(host) == 'On':
        print('当前已开机')
        return True
    ret = set_power_state(host, 'On')
    if ret:
        print('正在开机')

def poweroff(host, force=''):
    if get_power_state(host) != 'On':
        print('当前机器没有开机')
        return True
    state = 'ForceOff' if force else 'GracefulShutdown'
    ret = set_power_state(host, state)
    if ret:
        print('正在关机')


def main():
    args = parse_args(sys.argv[1:])
    if args.host == 'all':
        hosts = SERVERS.keys()
    else:
        hosts = [args.host]

    load_token()

    actions = {
        'status': get_status,
        'poweron': poweron,
        'poweroff': poweroff,
    }
    for host in hosts:
        if args.operator not in actions:
            continue
        print('-' * 30, host, '-' * 30)
        if login(host, username, password):
            actions[args.operator](host, args.arg)

    save_token()

if __name__ == '__main__':
    main()
