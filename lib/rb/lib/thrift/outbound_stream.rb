module Thrift
  class TOutboundStream < TBaseStream
    def initialize(iprot, oprot, name, seqid, klass, message_type, unlock = nil)
      @message_type = message_type
      @klass = klass
      @unlock = unlock
      super(iprot, oprot, name, seqid)
    end

    def ready
      Thread.new { read_goaway }
      super
    end

    def close
      wait_ready

      write_shell @message_type + 1

      @mutex.synchronize do
        loop do
          @cond.wait(@mutex)

          return nil if @closed
        end
      end
    end

    def send(arg)
      raise EOFError if @closed

      unless @oprot.trans.open?
        mark_as_closed
        raise EOFError
      end

      wait_ready
      write @message_type, @klass.new(arg: arg)
    end

    private

    def read_goaway
      mtype = read_shell

      write_shell @message_type + 2 if mtype == (@message_type + 1)
    ensure
      mark_as_closed
      @unlock&.call
    end
  end
end
