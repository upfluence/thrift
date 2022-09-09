module Thrift
  class TBaseStream
    def initialize(iprot, oprot, name, seqid)
      @iprot = iprot
      @oprot = oprot
      @name = name
      @seqid = seqid
      @closed = false
      @ready = false
      @mutex = Mutex.new
      @cond = ConditionVariable.new
    end

    def ready
      @ready = true
      @cond.broadcast
    end

    protected

    def wait_ready
      return if @ready || @closed

      @mutex.synchronize do
        loop do
          return if @ready || @closed

          @cond.wait(@mutex)
        end
      end
    end

    def read_message_begin
      name, mtype, seqid = @iprot.read_message_begin

      raise ApplicationException.new(
        ApplicationException::BAD_SEQUENCE_ID,
        'out of seq'
      ) if seqid != @seqid


      raise ApplicationException.new(
        ApplicationException::WRONG_METHOD_NAME,
        'wrong method name'
      ) if !name.nil? && name != @name

      mtype
    end

    def read_shell
      mtype = read_message_begin
      @iprot.read_message_end

      mtype
    end

    def write_unlocked(mtype, msg)
      @oprot.write_message_begin(@name, mtype, @seqid)
      msg.write(@oprot) if msg
      @oprot.write_message_end
      @oprot.trans.flush
    end

    def write(mtype, msg)
      @mutex.synchronize do
        write_unlocked(mtype, msg) unless @closed
      end
    end

    def write_shell_unlocked(mtype)
      write_unlocked(mtype, nil) unless @closed
    end

    def write_shell(mtype)
      write(mtype, nil)
    end

    def mark_as_closed
      @mutex.synchronize do
        @closed = true
        @cond.broadcast
      end
    end
  end
end
