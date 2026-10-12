module Thrift
  # How long, in seconds, a stream waits once it started to close for the other
  # side to answer: nothing of what is left on the connection can be trusted
  # after it, so the stream is aborted, and its connection with it. A value of
  # zero or less disables it.
  DEFAULT_STREAM_HANDSHAKE_TIMEOUT = 5

  # How long, in seconds, a stream can neither receive nor send a message
  # before it is aborted. A value of zero or less, the default, disables it, as
  # a quiet stream is a legitimate one.
  DEFAULT_STREAM_IDLE_TIMEOUT = 0

  class TBaseStream
    def initialize(
      iprot, oprot, name, seqid, handshake_timeout: nil, idle_timeout: nil
    )
      @iprot = iprot
      @oprot = oprot
      @name = name
      @seqid = seqid
      @closed = false
      @ready = false
      @mutex = Mutex.new
      @cond = ConditionVariable.new

      @handshake_timeout = handshake_timeout || DEFAULT_STREAM_HANDSHAKE_TIMEOUT
      @idle_timeout = idle_timeout || DEFAULT_STREAM_IDLE_TIMEOUT
      @aborted = false
      @last_activity = now
      @handshakes = []
      @watcher = nil
      @watch_mutex = Mutex.new
      @watch_cond = ConditionVariable.new
    end

    def ready
      @ready = true
      @cond.broadcast
      start_idle
    end

    def aborted?
      @aborted
    end

    protected

    def now
      Process.clock_gettime(Process::CLOCK_MONOTONIC)
    end

    def touch
      @last_activity = now
    end

    # start_idle aborts the stream once it did not receive nor send a message
    # for the idle timeout.
    def start_idle
      return unless @idle_timeout.positive?

      touch
      start_watcher
    end

    # start_handshake aborts the stream if the block does not return true once
    # the timeout elapsed: it is called when a go away is sent, which the other
    # side has to acknowledge.
    def start_handshake(&done)
      return unless @handshake_timeout.positive?

      @watch_mutex.synchronize do
        @handshakes << [now + @handshake_timeout, done]
      end

      start_watcher
      wake_watcher
    end

    # wake_watcher has to be called when the stream is closed, so that its
    # watcher stops.
    def wake_watcher
      @watch_mutex.synchronize { @watch_cond.broadcast }
    end

    def start_watcher
      @watch_mutex.synchronize do
        @watcher ||= Thread.new { watch }
      end
    end

    # The first deadline of the stream, nil if there is none.
    def next_deadline
      @handshakes.reject! { |(_, done)| done.call }

      deadlines = @handshakes.map(&:first)
      deadlines << @last_activity + @idle_timeout if @idle_timeout.positive?
      deadlines.min
    end

    # watch sleeps until the first deadline, and goes back to sleep if it moved
    # in the meantime: the stream was used, or a handshake was acknowledged.
    def watch
      @watch_mutex.synchronize do
        loop do
          return if @closed

          deadline = next_deadline

          if deadline.nil?
            @watch_cond.wait(@watch_mutex)
          else
            remaining = deadline - now

            break unless remaining.positive?

            @watch_cond.wait(@watch_mutex, remaining)
          end
        end
      end

      abort_stream
    rescue StandardError
      nil
    end

    # abort_stream closes the stream that ended because of an error, and drops
    # its connection: it can not be trusted anymore.
    def abort_stream
      return if @aborted

      @aborted = true

      [@iprot, @oprot].uniq.each do |prot|
        begin
          prot.trans.close
        rescue StandardError
          nil
        end
      end

      mark_as_closed
      on_abort
      @unlock&.call
    end

    def on_abort; end

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

      touch

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
      touch
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

      wake_watcher
    end
  end
end
