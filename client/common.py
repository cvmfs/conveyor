import gzip
import json
import os
import subprocess
import uuid

from base64 import b64encode
from contextlib import ContextDecorator


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
    lease_path = path
    if lease_path[0] != '/':
        lease_path = '/' + lease_path
    desc = {'repo': repo, 'payload': payload, 'path': lease_path, 'id': job_id}

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


class CvmfsTransaction(ContextDecorator):
    def __init__(self, job):
        self.repo = job['repo']
        self.job_id = job['id']
        self.full_path = self.repo
        if job['path'] != '/':
            self.full_path += job['path']

    def __enter__(self):
        print('-- Running CVMFS transaction for job {}'.format(self.job_id))
        subprocess.run(['cvmfs_server', 'transaction', self.full_path], check=True)
        return self

    def __exit__(self, *exc):
        if exc.count(None) == 3:
            print('-- Publishing CVMFS transaction for job {}'.format(self.job_id))
            subprocess.run(['cvmfs_server', 'publish', self.repo], check=True)
        else:
            print('-- Aborting CVMFS transaction for job {}'.format(self.job_id))
            subprocess.run(['cvmfs_server', 'abort', '-f', self.repo], check=True)

    def abort(self):
        raise RuntimeError('Aborting CVMFS transaction')
