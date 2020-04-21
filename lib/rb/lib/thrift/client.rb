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
#

module Thrift
  module Client
    def initialize(iprot, middlewares = [], oprot = nil)
      @iprot = iprot
      @oprot = oprot || iprot
      @middleware = case middlewares.length
                    when 0
                      Middleware::NOP_MIDDLEWARE
                    when 1
                      middlewares.first
                    else
                      Middleware::MultiMiddleware.new(middlewares)
                    end
      @seqid = 0
    end

    def send_message(name, args_class, args = {})
      @seqid += 1
      @oprot.write_message_begin(name, MessageTypes::CALL, @seqid)
      send_message_args(args_class, args)
    end

    def send_oneway_message(name, args_class, args = {})
      @seqid += 1
      @oprot.write_message_begin(name, MessageTypes::ONEWAY, @seqid)
      send_message_args(args_class, args)
    end

    def send_message_args(args_class, args)
      data = args_class.new
      args.each do |k, v|
        data.send("#{k.to_s}=", v)
      end
      begin
        data.write(@oprot)
      rescue StandardError => e
        @oprot.trans.close
        raise e
      end
      @oprot.write_message_end
      @oprot.trans.flush
    end

    def receive_message(result_klass, name=nil)
      fname, mtype, seqid = @iprot.read_message_begin
      handle_exception(mtype)

      raise ApplicationException.new(
        ApplicationException::BAD_SEQUENCE_ID,
        'out of seq'
      ) if seqid != @seqid

      raise ApplicationException.new(
        ApplicationException::INVALID_MESSAGE_TYPE,
        'invalid message type'
      ) if mtype != MessageTypes::REPLY

      raise ApplicationException.new(
        ApplicationException::WRONG_METHOD_NAME,
        'wrong method name'
      ) if !name.nil? && name != fname

      result = result_klass.new
      result.read(@iprot)
      @iprot.read_message_end
      result
    end

    def handle_exception(mtype)
      return if mtype != MessageTypes::EXCEPTION

      x = ApplicationException.new
      x.read(@iprot)
      @iprot.read_message_end
      raise x
    end
  end
end
