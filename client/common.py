import os

def get_rmq_creds():
    rmq_url = os.getenv('CVMFS_RMQ_URL')
    rmq_username = os.getenv('CVMFS_RMQ_USERNAME')
    rmq_password = os.getenv('CVMFS_RMQ_PASSWORD')
    rmq_port = os.getenv('CVMFS_RMQ_PORT')
    if rmq_port == None or rmq_port == '':
        rmq_port = '5672'

    rmq_vhost = os.getenv('CVMFS_RMQ_VHOST')
    if rmq_vhost == None or rmq_vhost == '':
        rmq_vhost = '/cvmfs'

    if ((rmq_url == '') or
        (rmq_url == None) or
        (rmq_username == '') or
        (rmq_username == None) or
        (rmq_password == '') or
        (rmq_password == None)):
        return None

    return {'url' : rmq_url,
            'username' : rmq_username,
            'password' : rmq_password,
            'port' : int(rmq_port),
            'vhost' : rmq_vhost}