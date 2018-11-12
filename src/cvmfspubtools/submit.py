import json
import pika

from cvmfspubtools.config import constants
from cvmfspubtools.jobs import create_description
from pprint import pprint


def add_submit_arguments(subparsers):
    submit_parser = subparsers.add_parser('submit', help='Submit a jobs')
    submit_parser.add_argument('repo', help='repository name')
    submit_parser.add_argument('payload', help='payload URL')
    submit_parser.add_argument('-p', '--path', default='/',
                               help='target path in repository')
    submit_parser.add_argument(
        '-c', '--config', default='/etc/cvmfs/publisher/config.json', help='config file')
    submit_parser.add_argument(
        '-s', '--script', help='script to run during transaction')
    submit_parser.add_argument(
        '--script-args', help='arguments for the transaction script')
    submit_parser.add_argument(
        '--remote-script', action='store_true', help='transaction script is a remote file')
    submit_parser.add_argument(
        '-d', '--deps', help='Comma-separated list of IDs of job dependencies')


def submit_job(rabbitmq_config, arguments):
    credentials = pika.PlainCredentials(rabbitmq_config['username'],
                                        rabbitmq_config['password'])

    parameters = pika.ConnectionParameters(rabbitmq_config['url'],
                                           rabbitmq_config['port'],
                                           rabbitmq_config['vhost'],
                                           credentials)

    connection = pika.BlockingConnection(parameters)
    channel = connection.channel()

    channel.exchange_declare(exchange=constants['new_job_exchange'],
                             exchange_type='direct',
                             durable=True)

    job_description = create_description(arguments.repo,
                                         arguments.payload,
                                         arguments.path,
                                         script=arguments.script,
                                         script_args=arguments.script_args,
                                         remote_script=arguments.remote_script,
                                         deps=arguments.deps)

    print('Job description:')
    pprint(job_description)

    msg = json.dumps(job_description)

    channel.basic_publish(exchange=constants['new_job_exchange'],
                          routing_key=constants['routing_key'],
                          body=msg,
                          properties=pika.BasicProperties(delivery_mode=2))

    print('Result:')
    result = {'status': 'ok', 'job_id': job_description['id']}
    print(json.dumps(result))
