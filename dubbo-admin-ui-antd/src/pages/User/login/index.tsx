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
import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, message } from 'antd';
import React, { useState } from 'react';
import ProForm, { ProFormCheckbox, ProFormText } from '@ant-design/pro-form';
import { useIntl, Link, history, FormattedMessage, SelectLang } from 'umi';
import Footer from '@/components/Footer';
import { login } from '@/services/ant-design-pro/login';

import styles from './index.less';

const LoginMessage: React.FC<{
  content: string;
}> = ({ content }) => (
  <Alert
    style={{
      marginBottom: 24,
    }}
    message={content}
    type="error"
    showIcon
  />
);

/** This method jumps to the position of the redirect parameter */
const goto = () => {
  if (!history) return;
  setTimeout(() => {
    const { query } = history.location;
    const { redirect } = query as { redirect: string };
    history.push(redirect || '/');
  }, 10);
};

const Login: React.FC = () => {
  const [submitting, setSubmitting] = useState(false);
  const [userLoginState, setUserLoginState] = useState<API.LoginResult>({});
  const [type] = useState<string>('account');

  const intl = useIntl();

  const handleSubmit = async (values: API.LoginParams) => {
    setSubmitting(true);
    try {
      // login
      const msg = await login({ ...values, type });
      if (msg !== '') {
        message.success(
          intl.formatMessage({
            id: 'pages.login.accountLogin.successful',
            defaultMessage: 'Login successful!',
          }),
        );
        localStorage.setItem('token', msg);
        localStorage.setItem('username', values.username);
        goto();
        return;
      }
      // If it fails, set the user error message
      setUserLoginState(msg);
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'pages.login.accountLogin.error',
          defaultMessage: 'Login failed, please try again!',
        }),
      );
    }
    setSubmitting(false);
  };
  const { status } = userLoginState;

  return (
    <div className={styles.container}>
      <div className={styles.lang} style={{ marginTop: '100px' }}>
        {SelectLang && <SelectLang />}
      </div>
      <div className={styles.content}>
        <div className={styles.top}>
          <div className={styles.header}>
            <Link to="/">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 321.39 78.54"
                style={{ height: '32px' }}
              >
                <title>DUBBO LOGO</title>
                <path
                  d="M68.46 50.38c0 14.06 11.39 22.11 25.45 22.11s25.45-8.05 25.45-22.11V7.25H68.46zm21.24-28h8.6V31H89.7zm0 22.25h8.6v8.6H89.7zM33.24 7.25H4.06v64H33.24c10.95.0 19.3-7.18 23.29-17.15a45.12 45.12.0 002.38-14.87A45.12 45.12.0 0056.53 24.4C52.84 14.62 44.19 7.25 33.24 7.25zm.43 14.63H30.34a3.44 3.44.0 00-3.44 3.44V53.23a3.44 3.44.0 003.44 3.44h3.33v4.63h-8.3a6.87 6.87.0 01-6.87-6.87V24.12a6.87 6.87.0 016.87-6.87h8.3zM285.51 6.06c-17.05.0-30.88 10.28-30.88 33.21s13.83 33.21 30.88 33.21 30.88-10.28 30.88-33.21S302.56 6.06 285.51 6.06zm7.59 48.36a6.87 6.87.0 01-6.87 6.87h-8.3V56.67h3.33a3.44 3.44.0 003.44-3.44V25.31a3.44 3.44.0 00-3.44-3.44h-3.33V17.25h8.3a6.87 6.87.0 016.87 6.87zm-53.4-17.56A17.39 17.39.0 00227.31 7.25H195.1v64h32.21a19.44 19.44.0 0012.38-34.44zM211.63 61.29h-6.08l18.68-44h6.08zM177 36.85A17.39 17.39.0 00164.65 7.25H132.43v64h32.21A19.44 19.44.0 00177 36.85zM149 61.29h-6.08l18.68-44h6.08z"
                  style={{ fill: '#1a8cf7', fillOpacity: 1 }}
                ></path>
              </svg>
              <span className={styles.title}> Admin</span>
            </Link>
          </div>
        </div>

        <div className={styles.main}>
          <ProForm
            initialValues={{
              autoLogin: true,
            }}
            submitter={{
              searchConfig: {
                submitText: intl.formatMessage({
                  id: 'pages.login.submit',
                  defaultMessage: 'Login',
                }),
              },
              render: (_, dom) => dom.pop(),
              submitButtonProps: {
                loading: submitting,
                size: 'large',
                style: {
                  width: '100%',
                },
              },
            }}
            onFinish={async (values) => {
              handleSubmit(values as API.LoginParams);
            }}
          >
            <div style={{ height: '30px' }}></div>

            {status === 'error' && (
              <LoginMessage
                content={intl.formatMessage({
                  id: 'pages.login.accountLogin.errorMessage',
                  defaultMessage: 'Incorrect username/password',
                })}
              />
            )}
            <>
              <ProFormText
                name="username"
                fieldProps={{
                  size: 'large',
                  prefix: <UserOutlined className={styles.prefixIcon} />,
                }}
                placeholder={intl.formatMessage({
                  id: 'pages.login.username.placeholder',
                  defaultMessage: 'Username: admin or user',
                })}
                rules={[
                  {
                    required: true,
                    message: (
                      <FormattedMessage
                        id="pages.login.username.required"
                        defaultMessage="Please input your username!"
                      />
                    ),
                  },
                ]}
              />
              <ProFormText.Password
                name="password"
                fieldProps={{
                  size: 'large',
                  prefix: <LockOutlined className={styles.prefixIcon} />,
                }}
                placeholder={intl.formatMessage({
                  id: 'pages.login.password.placeholder',
                  defaultMessage: 'Default password: root',
                })}
                rules={[
                  {
                    required: true,
                    message: (
                      <FormattedMessage
                        id="pages.login.password.required"
                        defaultMessage="Please input your password!"
                      />
                    ),
                  },
                ]}
              />
            </>

            <div
              style={{
                marginBottom: 24,
              }}
            >
              <ProFormCheckbox noStyle name="autoLogin">
                <FormattedMessage id="pages.login.rememberMe" defaultMessage="Remember me" />
              </ProFormCheckbox>
            </div>
          </ProForm>
        </div>
      </div>
      <Footer />
    </div>
  );
};

export default Login;
