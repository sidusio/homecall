import { StyleSheet } from 'react-native';
import * as SecureStore from 'expo-secure-store';
import { WebView } from 'react-native-webview';
import { Text } from 'react-native';
import { useState, useEffect } from 'react';

export default function Call(props: {
  instanceUrl: string,
  token: string,
  settings: any,
  deviceId: string,
}){
  const { token, instanceUrl } = props;

  const [webViewRef, setWebViewRef] = useState(null);

  const injectToken = async () => {
    if (!webViewRef) {
      return;
    }

    const jsCode = `
      window.localStorage.setItem('token', '${token}');
    `;

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

    webViewRef.injectJavaScript(debugging);
  }

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
    <WebView
      style={styles.container}
      source={{ uri: `${fixInstanceUrl()}/device` }}
      ref={setWebViewRef}
      onLoad={initialLoad}
      onMessage={onMessage}
      mediaPlaybackRequiresUserAction={ false }
      allowsInlineMediaPlayback={ true }
    />
  )
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
  },
});
