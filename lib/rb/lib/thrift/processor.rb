#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements. See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership. The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License. You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.
#

module Thrift
  module Processor
    def initialize(handler, middlewares = [])
      @handler = handler
      @middleware = Middleware.wrap(middlewares)
    end

    def process(iprot, oprot)
      name, _type, seqid = iprot.read_message_begin
      if respond_to?("process_#{name}")
        begin
          send("process_#{name}", seqid, iprot, oprot)
        rescue => e
          write_exception(e, oprot, name, seqid)
        end
        true
      else
        iprot.skip(Types::STRUCT)
        iprot.read_message_end
        write_exception(
          ApplicationException.new(
            ApplicationException::UNKNOWN_METHOD,
            'Unknown function ' + name,
          ),
          oprot,
          name,
          seqid
        )
        false
      end
    end

    def read_args(iprot, args_class)
      args = args_class.new
      args.read(iprot)
      iprot.read_message_end
      args
    end

    def write_exception(exception, oprot, name, seqid)
      oprot.write_message_begin(name, MessageTypes::EXCEPTION, seqid)

      unless exception.is_a? ApplicationException
        exception = ApplicationException.new(
          ApplicationException::INTERNAL_ERROR,
          "Internal error processing #{name}: #{exception.class}: #{exception}"
        )
      end

      exception.write(oprot)
      oprot.write_message_end
      oprot.trans.flush
    end

    def write_result(result, oprot, name, seqid)
      oprot.write_message_begin(name, MessageTypes::REPLY, seqid)
      result.write(oprot)
      oprot.write_message_end
      oprot.trans.flush
    end
  end
end
