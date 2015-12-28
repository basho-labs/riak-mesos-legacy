#
#    Copyright (C) 2015 Basho Technologies, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
from setuptools import setup, find_packages
from codecs import open
from os import path

from dcos_riak import constants


here = path.abspath(path.dirname(__file__))

# Get the long description from the relevant file
with open(path.join(here, 'README.rst'), encoding='utf-8') as f:
    long_description = f.read()

setup(
    name='dcos-riak',

    # Versions should comply with PEP440.  For a discussion on single-sourcing
    # the version across setup.py and the project code, see
    # https://packaging.python.org/en/latest/single_source_version.html
    version=constants.version,

    description='DCOS Riak Command Line Interface',
    long_description=long_description,

    # The project's main homepage.
    url='https://github.com/basho-labs/riak-mesos',

    # Author details
    author='Basho Technologies, Inc.',
    author_email='support@basho.com',

    classifiers=[
        'Development Status :: 3 - Beta',
        'Intended Audience :: Developers',
        'Intended Audience :: Information Technology',
        'License :: OSI Approved :: TODO: License',
        'Programming Language :: Python :: 2',
        'Programming Language :: Python :: 2.6',
        'Programming Language :: Python :: 2.7',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.2',
        'Programming Language :: Python :: 3.3',
        'Programming Language :: Python :: 3.4',
    ],
    keywords='dcos command riak mesosphere',
    packages=find_packages(exclude=['contrib', 'docs', 'tests*']),
    install_requires=[
        'dcos>=0.1.6, <1.0',
        'docopt',
        'toml',
        'requests',
        'six>=1.9, <2.0'
    ],
    extras_require={
        'dev': ['check-manifest'],
        'test': ['coverage'],
    },
    package_data={},
    # package_data={'dcos_riak': ['zktool_darwin_amd64','zktool_linux_amd64']},
    # include_package_data=True,
    # data_files=[],
    data_files=[('dcos_riak', ['zktool_darwin_amd64','zktool_linux_amd64'])]

    # To provide executable scripts, use entry points in preference to the
    # "scripts" keyword. Entry points provide cross-platform support and allow
    # pip to create the appropriate form of executable for the target platform.
    entry_points={
        'console_scripts': [
            'dcos-riak=dcos_riak.cli:main',
        ],
    },
)
