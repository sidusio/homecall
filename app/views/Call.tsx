import { StyleSheet } from 'react-native';
import { WebView } from 'react-native-webview';
import { useState, useEffect } from 'react';

export default function Call(props: {
  instanceUrl: string,
  token: string,
  settings: any,
  deviceId: string,
}){
  const { token, instanceUrl } = props;

  const [webViewRef, setWebViewRef] = useState(null);
  const [lastRefresh, setLastRefresh] = useState(Date.now());

  /**
   * Injects the token into the WebView.
   *
   * @returns void
   */
  const injectToken = (): void => {
    if (!webViewRef) {
      return;
    }

    const jsCode = `
      window.localStorage.setItem('token', '${token}');
    `;

    webViewRef.injectJavaScript(jsCode);
  }

  /**
   * Injects the device data into the WebView.
   *
   * @returns void
   */
  const injectData = (): void => {
    if (!webViewRef) {
      return;
    }

    const reactNativeData = {
      settings: props.settings,
      deviceId: props.deviceId,
    };

    const stringifyReactNativeData = JSON.stringify(reactNativeData); // Data has to be stringified to be injected.

    const jsCode = `
      window.deviceData = ${stringifyReactNativeData};
    `;

    webViewRef.injectJavaScript(jsCode);
  }

  /**
   * Injects debugging code into the WebView.
   *
   * @returns void
   */
  const injectDebugging = (): void => {
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

  /**
   * Refreshes the WebView.
   *
   * @returns void
   */
  const refresh = (): void => {
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

  /**
   * Handles messages from the WebView.
   *
   * @param event - The message event
   */
  const onMessage = (event: any) => {
    const message = event.nativeEvent.data;
    console.log('WebView: ' + message);
  }

  /**
   * Initial load of the WebView.
   *
   * @returns void
   */
  const initialLoad = (): void => {
    injectToken();
    injectData();
    injectDebugging();
  }

  // All the useEffects.
  useEffect(() => {
    injectToken();
  }, [token, webViewRef]);

  useEffect(() => {
    injectDebugging();
  }, [webViewRef]);

  useEffect(() => {
    const interval = setInterval(refresh, 30 * 60 * 1000);
    return () => clearInterval(interval);
  }, [lastRefresh]);

  return (
    <WebView
      style={styles.container}
      source={{ uri: `${instanceUrl}/device` }}
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
