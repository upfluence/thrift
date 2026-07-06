/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements. See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership. The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

namespace go streamingtest
namespace rb Thrift.StreamingTest

/**
 * Cross-language streaming test service.
 *
 * Three methods exercise each streaming RPC type:
 *
 *  testClientStream  - client-side streaming (sink):
 *    The client sends N strings into the sink and then closes it.
 *    The server returns the number of messages it received.
 *
 *  testServerStream  - server-side streaming (stream):
 *    The client sends a count and a prefix string.
 *    The server pushes `count` strings of the form "{prefix}_{i}" and closes.
 *
 *  testBidi          - bidirectional streaming (stream + sink):
 *    The client sends strings into the sink; the server echoes each one
 *    back uppercased through the stream.  The server closes its stream
 *    after receiving the client's GoAway.
 */
service StreamingTest {
  /**
   * Client streams N strings; server acknowledges (void return).
   */
  void, sink<string> testClientStream(1: string label),

  /**
   * Server streams `count` strings prefixed with `prefix`.
   */
  void, stream<string> testServerStream(1: i32 count, 2: string prefix),

  /**
   * Bidirectional: client sends strings, server echoes them uppercased.
   */
  void, stream<string>, sink<string> testBidi(1: string label),
}
