import gzip
import uuid

from base64 import b64encode


def create_description(repo, payload, path, **kwargs):
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

