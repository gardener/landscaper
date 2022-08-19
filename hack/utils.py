# SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
#
# SPDX-License-Identifier: Apache-2.0

import os
import tempfile
import base64
import subprocess


class TempFileAuto(object):
    def __init__(self, prefix=None, mode='w+', suffix=".yaml"):
        self.file_obj = tempfile.NamedTemporaryFile(mode=mode, prefix=prefix, suffix=suffix, delete=False)
        self.name = self.file_obj.name
    def __enter__(self):
        return self
    def write(self, b):
        self.file_obj.write(b)
    def writelines(self, lines):
        self.file_obj.writelines(lines)
    def switch(self):
        self.file_obj.close()
        return self.file_obj.name
    def __exit__(self, type, value, traceback):
        if not self.file_obj.closed:
            self.file_obj.close()
        os.remove(self.file_obj.name)
        return False

def base64_encode_to_string(value):
    value = value.rstrip()
    if isinstance(value, str):
        value = value.encode('utf-8')
    return base64.b64encode(value).decode('utf-8')
