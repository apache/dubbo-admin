/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package eventbus

type Config struct {
	// BufferSize controls the buffer for every single event listener.
	// If we go over buffer, additional delay may happen to various operation like insight recomputation or DDS.
	BufferSize uint `json:"bufferSize" envconfig:"DUBBO_EVENT_BUS_BUFFER_SIZE"`
}

func (c Config) Validate() error {
	return nil
}

func Default() Config {
	return Config{
		BufferSize: 100,
	}
}
