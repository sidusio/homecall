import { StyleSheet } from 'react-native';
import { WebView } from 'react-native-webview';
import { Text, View, Pressable } from 'react-native';
import { useState, useEffect } from 'react';
import messaging from '@react-native-firebase/messaging';
import { deviceClient } from './../services/api';
import { clearCredentials } from "../services/auth";

export default function Call(props: {
  instanceUrl: string,
  token: string,
  settings: any,
  deviceId: string,
}){
  const { token, instanceUrl } = props;

  const [webViewRef, setWebViewRef] = useState(null);

  useEffect(() => {
    if(!token || !instanceUrl) {
      return;
    }

    messaging().requestPermission();

    messaging()
      .getToken()
      .then(token => {
        deviceClient(instanceUrl).updateNotificationToken({ notificationToken: token }, {
          headers: {
            Authorization: `Bearer ${props.token}`,
          }
        })
      })
  }, [token, instanceUrl]);

  useEffect(() => {
    const unsubscribe = messaging().onMessage(async remoteMessage => {
      // Inject the message into the webview
      if (webViewRef) {
        const stringifiedMessage = JSON.stringify(remoteMessage);

        const jsCode = `
          window.dispatchEvent(new CustomEvent('fcm', { detail: ${stringifiedMessage} }));
        `;

        // @ts-ignore
        webViewRef.injectJavaScript(jsCode);
      }
    });

    return unsubscribe;
  });

  const injectToken = async () => {
    if (!webViewRef) {
      return;
    }

    const jsCode = `
      window.localStorage.setItem('token', '${token}');
    `;

    // @ts-ignore
    webViewRef.injectJavaScript(jsCode);
  }

  const injectData = async () => {
    if (!webViewRef) {
      return;
    }

    const reactNativeData = {
      settings: props.settings,
      deviceId: props.deviceId,
    };

    const stringifyReactNativeData = JSON.stringify(reactNativeData);

    const jsCode = `
      window.deviceData = ${stringifyReactNativeData};
    `;

    // @ts-ignore
    webViewRef.injectJavaScript(jsCode);
  }

  useEffect(() => {
    injectToken();
  }, [token, webViewRef]);

  const [lastRefresh, setLastRefresh] = useState(Date.now());

  const refresh = () => {
    if (!webViewRef) {
      return;
    }

    // if refreshed in the last 6 hours, don't refresh again
    if (lastRefresh > Date.now() - 6 * 60 * 60 * 1000) {
      return;
    }

    // Only refresh between 1am and 4am
    const now = new Date();
    if (now.getHours() < 1 || now.getHours() > 4) {
      return;
    }

    // @ts-ignore
    webViewRef.reload();
    initialLoad();
    setLastRefresh(Date.now());
  }

  const injectDebugging = () => {
    if (!webViewRef) {
      return;
    }

    const debugging = `
     // Debug
     console = new Object();
     console.log = function(log) {
      window.ReactNativeWebView.postMessage(JSON.stringify(log))
     };
     console.debug = console.log;
     console.info = console.log;
     console.warn = console.log;
     console.error = console.log;
     `;

    // @ts-ignore
    webViewRef.injectJavaScript(debugging);
  }

  // @ts-ignore
  const onMessage = (event) => {
    const message = event.nativeEvent.data;
    console.log('WebView: ' + message);
  }

  const initialLoad = () => {
    injectToken();
    injectData();
    injectDebugging();
  }

  useEffect(() => {
    injectDebugging();
  }, [webViewRef]);

  useEffect(() => {
    const interval = setInterval(refresh, 30 * 60 * 1000);
    return () => clearInterval(interval);
  }, [lastRefresh]);

  // TODO: Make better fix than this.
  const fixInstanceUrl = () => {
    // remove '/api'
    if (instanceUrl.endsWith('/api')) {
      return instanceUrl.slice(0, -4);
    }
  }

  return (
    <>
      <Pressable
        style={styles.button}
        onPress={clearCredentials}
      >
        <Text>Avregistrera</Text>
      </Pressable>

      <WebView
        style={styles.container}
        source={{ uri: `${fixInstanceUrl()}/device` }}
        // @ts-ignore
        ref={setWebViewRef}
        onLoad={initialLoad}
        onMessage={onMessage}
        mediaPlaybackRequiresUserAction={ false }
        allowsInlineMediaPlayback={ true }
      />
    </>
  )
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
  },
  text: {
    alignItems: 'center',
    padding: 40,
    margin: 20,
    backgroundColor: '#DDDDDD',
    borderRadius: 10,
  },
  button: {
    position: 'absolute',
    zIndex: 1,
    bottom: 90,
    right: 10,
    padding: 20,
    backgroundColor: 'rgb(67, 107, 177)',
  }
});
