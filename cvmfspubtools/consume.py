import json
import os
import pika
import subprocess

from cvmfspubtools.config import constants, read_config
from cvmfspubtools.transaction import Transaction
from pathlib import Path
from shutil import rmtree
from urllib.parse import urlparse
from urllib.request import urlretrieve

# Global values (should be constant during run time)
temp_dir = '/tmp/cvmfs-consumer'


def add_consume_arguments(subparsers):
    consume_parser = subparsers.add_parser('consume', help='Consume jobs')

    consume_parser.add_argument(
        '-c', '--config',
        default='/etc/cvmfs/publisher/config.json',
        help='config file')
    consume_parser.add_argument(
        '--temp-dir',
        default='/tmp/cvmfs-consumer',
        help='temporary dir for use during CVMFS transaction'
    )

def callback(ch, method, properties, body):
    job = json.loads(body)
    print('-- Start publishing job {}'.format(job['id']))

    try:
        run_cvmfs_transaction(job)
        ch.basic_ack(delivery_tag=method.delivery_tag)
    except Exception as e:
        print('Exception raised during CVMFS transaction: {}'.format(e))
        # TODO: change the following to a resubmit with retry += 1
        ch.basic_nack(delivery_tag=method.delivery_tag,
                      multiple=False, requeue=True)

    print('-- Finished publishing job {}'.format(job['id']))


def run_cvmfs_transaction(job):
    with Transaction(job):
        # save payload into temp dir
        url = urlparse(job['payload'])
        payload_file_name = os.path.join(temp_dir, Path(url.path).name)
        payload_file_name, _ = urlretrieve(
            job['payload'], filename=payload_file_name)

        # untar payload into path dir
        target_dir = '/cvmfs/' + job['repo'] + '/' + job['path']
        os.makedirs(target_dir, exist_ok=True)
        subprocess.run(['tar', '-C', target_dir, '-xf',
                        payload_file_name], check=True)

        # (optional) unpack transaction script into tmp dir
        # run transaction script

    # cleanup payload temp file
    os.remove(payload_file_name)


def consume_jobs(rabbitmq_config, arguments):
    try:
        global temp_dir
        temp_dir = arguments.temp_dir
        os.makedirs(temp_dir, exist_ok=True)

        credentials = pika.PlainCredentials(rabbitmq_config['username'],
                                            rabbitmq_config['password'])
        parameters = pika.ConnectionParameters(rabbitmq_config['url'],
                                                rabbitmq_config['port'],
                                                rabbitmq_config['vhost'],
                                                credentials)
        connection = pika.BlockingConnection(parameters)
        channel = connection.channel()
        channel.basic_qos(prefetch_count=1)

        result = channel.queue_declare(queue=constants['new_job_queue'])
        queue = result.method.queue
        channel.queue_bind(exchange=constants['new_job_exchange'], routing_key='', queue=queue)
        channel.basic_consume(callback, queue=queue, no_ack=False)

        print('-- Waiting for jobs. To exit, press Ctrl-C')
        channel.start_consuming()
    finally:
        rmtree(temp_dir)
