module Thrift
  class TInboundStream < TBaseStream
    def initialize(iprot, oprot, name, seqid, klass, message_type, unlock = nil)
      @message_type = message_type
      @klass = klass
      @unlock = unlock
      @closing = false
      super(iprot, oprot, name, seqid)
    end

    def close
      wait_ready

      @mutex.synchronize do
        return nil if @closed

        begin
          @closing = true
          write_shell_unlocked @message_type + 1

          loop do
            mtype = read_message_begin

            case mtype
            when @message_type
              @iprot.skip(Types::STRUCT)
              @iprot.read_message_end
            when @message_type + 2
              @iprot.read_message_end
              return
            else
              raise ApplicationException.new(
                ApplicationException::INVALID_MESSAGE_TYPE,
                'invalid message type'
              )
            end
          end
        ensure
          @unlock&.call
        end
      end
    end

    def receive
      wait_ready

      @mutex.synchronize do
        case read_message_begin
        when @message_type
          res = @klass.new
          res.read(@iprot)
          @iprot.read_message_end

          return res.arg
        when @message_type + 1
          @iprot.read_message_end
          write_shell_unlocked @message_type + 2 unless @closing
          @closed = true
          @cond.broadcast
          @unlock&.call

          raise EOFError
        else
          raise ApplicationException.new(
            ApplicationException::INVALID_MESSAGE_TYPE,
            'invalid message type'
          )
        end
      end
    end
  end
end
