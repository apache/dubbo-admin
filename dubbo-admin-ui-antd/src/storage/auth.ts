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
/* eslint no-useless-escape:0 import/prefer-default-export:0 */

 /**
  * token key
  */
  const tokenKey = 'token';

  /**
   * username token
   */
  const usernameKey = 'usernameKey';

  export const setToken = (token:string)=>{
     localStorage.setItem(tokenKey,token);
  }

  export const getToken = (): string => {
    var result = localStorage.getItem(tokenKey);
    return result?result:'';
  }

  export const removeToken = ()=>{
    localStorage.removeItem(tokenKey);
  }

  export const setUsername = (username:string) => {
     localStorage.setItem(usernameKey,username);
  }

  export const getUsername = ():string =>{
    var result = localStorage.getItem(usernameKey);
    return result?result:'';
  }

  export const removeUsername = ()=>{
    localStorage.removeItem(usernameKey);
  }
