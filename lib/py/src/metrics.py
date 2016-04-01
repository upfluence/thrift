import os
import datetime
import tornado.gen
import statsd

if os.environ.get("STATSD_URL", None):
    ip, port = os.environ.get("STATSD_URL").split(':')
    statsd_client = statsd.StatsClient(host=ip, port=int(port),
                                       prefix=os.environ.get('STATSD_PREFIX',
                                                             None))
else:
    statsd_client = statsd.StatsClient()


def instrument(name):
    def instrument_wrapper(fn):
        @tornado.gen.coroutine
        def fn_wrapper(*args, **kwargs):
            def send_duration():
                duration = (datetime.datetime.now() - start).total_seconds()
                statsd_client.timing("{}.duration".format(name),
                                     int(duration * 1000))

            start = datetime.datetime.now()
            ftr_result = fn(*args, **kwargs)

            try:
                result = yield tornado.gen.maybe_future(ftr_result)
            except Exception as e:
                if statsd_client:
                    statsd_client.incr("{}.exceptions.{}".format(
                        name, e.__class__.__name__.lower()))
                    send_duration()

                raise e
            else:
                if statsd_client:
                    send_duration()
                    statsd_client.incr("{}.success".format(name))

                raise tornado.gen.Return(result)

        return fn_wrapper
    return instrument_wrapper
