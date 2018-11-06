import gzip
import json
import os
import uuid

from base64 import b64encode
from contextlib import contextmanager

def read_config(config_file):
    with open(config_file) as f:
        cfg = json.load(f)

    if 'port' not in cfg['rabbitmq']:
        cfg['rabbitmq']['port'] = 5672

    if 'vhost' not in cfg['rabbitmq']:
        cfg['rabbitmq']['vhost'] = '/cvmfs'

    return cfg


def create_job_description(repo, payload, path, **kwargs):
    job_id = str(uuid.uuid1())
    desc = {'repo': repo, 'payload': payload, 'path': path, 'id': job_id}

    if 'script' in kwargs and kwargs['script'] is not None:
        desc['remote_script'] = kwargs['remote_script']
        if kwargs['remote_script']:
            desc['script'] = kwargs['script']
        else:
            with open(kwargs['script'], 'rb') as f:
                desc['script'] = b64encode(
                    gzip.compress(f.read())).decode('utf-8')

    if 'deps' in kwargs and kwargs['deps'] is not None:
        deps = kwargs['deps'].split(',')
        desc['deps'] = deps

    return desc

@contextmanager
def cvmfs_transaction(job):
    print(job)
    full_path = job['repo']
    if job['path'] != '/':
        full_path += job['path']
    print("cvmfs_server transaction {}".format(full_path))
    yield full_path
    print("cvmfs_server abort {}".format(job['repo']))
