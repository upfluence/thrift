import tornado.testing
import tornado.gen
import os

os.environ = {}

import metrics
import mock


@metrics.instrument("test_coroutine")
@tornado.gen.coroutine
def success_coroutine():
    yield tornado.gen.sleep(1)
    raise tornado.gen.Return("foo")


@metrics.instrument("test_coroutine")
@tornado.gen.coroutine
def exception_coroutine():
    raise StandardError()


class MetricsTestCase(tornado.testing.AsyncTestCase):
    def setUp(self):
        super(MetricsTestCase, self).setUp()

        metrics.statsd_client = mock.Mock()
        metrics.statsd_client.timing = mock.Mock()
        metrics.statsd_client.incr = mock.Mock()

    @tornado.testing.gen_test
    def test_success_method(self):
        yield success_coroutine()

        arg1, arg2 = metrics.statsd_client.timing.call_args_list[0][0]
        self.assertEqual(arg1, "test_coroutine.duration")
        self.assertIn(arg2, xrange(1000, 1200))

        metrics.statsd_client.incr.assert_called_with("test_coroutine.success")

    @tornado.testing.gen_test
    def test_exception_method(self):
        try:
            yield exception_coroutine()

            assert False
        except StandardError:
            metrics.statsd_client.timing.assert_called_with(
                "test_coroutine.duration", 0)
            metrics.statsd_client.incr.assert_called_with(
                "test_coroutine.exceptions.standarderror")
