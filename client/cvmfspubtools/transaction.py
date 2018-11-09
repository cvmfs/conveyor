import subprocess

from contextlib import ContextDecorator

class Transaction(ContextDecorator):
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
