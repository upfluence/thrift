module Thrift
  class TBidiInboundStream
    def initialize(bidi_stream)
      @bidi_stream = bidi_stream
    end

    def receive
      @bidi_stream.receive
    end

    def close
      @bidi_stream.inbound_close
    end
  end

  class TBidiOutboundStream
    def initialize(bidi_stream)
      @bidi_stream = bidi_stream
    end

    def send(msg)
      @bidi_stream.send(msg)
    end

    def close
      @bidi_stream.outbound_close
    end
  end

  class TBidiStream < TBaseStream
    def initialize(iprot, oprot, name, seqid, inbound_klass, outbound_klass, inbound_message_type, outbound_message_type, unlock = nil)
      @inbound_message_type = inbound_message_type
      @outbound_message_type = outbound_message_type
      @inbound_klass = inbound_klass
      @outbound_klass = outbound_klass
      @inbound_closed = false
      @outbound_closed = false
      @message_queue = Queue.new
      @unlock = unlock
      @closing_inbound = false
      @closing_outbound = false
      super(iprot, oprot, name, seqid)
    end

    def ready
      Thread.new { background_read }
      super
    end

    def inbound_close
      wait_ready

      @mutex.synchronize do
        return if @closed || @inbbound_closed

        @closing_inbound = true
        write_shell_unlocked @inbound_message_type + 1

        loop do
          return if @closed || @inbound_closed

          @cond.wait(@mutex)
        end
      end
    end

    def outbound_close
      wait_ready

      @mutex.synchronize do
        return if @closed || @outbound_closed

        @closing_outbound = true
        write_shell_unlocked @outbound_message_type + 1

        loop do
          return nil if @closed || @outbound_closed

          @cond.wait(@mutex)
        end
      end
    end

    def receive
      raise EOFError if @closed || @inbound_closed

      msg = @message_queue.pop

      raise EOFError unless msg

      msg.arg
    end

    def send(msg)
      raise EOFError if @closed || @outbound_closed

      unless @oprot.trans.open?
        mark_as_closed
        raise EOFError
      end

      wait_ready

      write @outbound_message_type, @outbound_klass.new(arg: msg)
    end

    def close
      mark_as_closed
    end

    private

    def read_message
      msg = @inbound_klass.new
      msg.read(@iprot)
      @iprot.read_message_end

      @message_queue << msg unless @inbound_closed
    end

    def cleanup_inbound
      @mutex.synchronize do
        @inbound_closed = true
        @message_queue.close unless @message_queue.closed?
        @cond.broadcast
      end
    end

    def cleanup_outbound
      @mutex.synchronize do
        @outbound_closed = true
        @cond.broadcast
      end
    end

    def background_read
      loop do
        begin
          mtype = read_message_begin

          case mtype
          when @inbound_message_type
            read_message
          when @inbound_message_type + 1
            @mutex.synchronize do
              if !@inbound_closed && !@closing_inbound
                write_shell_unlocked @inbound_message_type + 2
              end
            end

            cleanup_inbound
          when @inbound_message_type + 2
            cleanup_inbound
          when @outbound_message_type + 1
            @mutex.synchronize do
              if !@outbound_closed && !@closing_outbound
                write_shell_unlocked @outbound_message_type + 2
              end
            end
            cleanup_outbound
          when @outbound_message_type + 2
            cleanup_outbound
          end
        rescue Exception => e
          mark_as_closed
          @message_queue.close unless @message_queue.closed?
          @cond.broadcast
        end

        if @outbound_closed && @inbound_closed
          @closed = true
          @cond.broadcast
        end

        if @closed
          @unlock&.call
          @cond.broadcast
          return
        end
      end
    end
  end
end
