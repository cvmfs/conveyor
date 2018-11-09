from setuptools import setup, find_packages

setup(
    name='cvmfspubtools',
    version='0.1',
    packages=find_packages(),

    install_requires=['pika'],

    scripts=['cvmfs_job'],

    author='Radu Popescu',
    author_email='radu.popescu@cern.ch',
    description="CernVM-FS publisher client tools",
    license='BSD',
)