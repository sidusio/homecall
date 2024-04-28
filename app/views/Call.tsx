import {StyleSheet} from 'react-native';
import { WebView } from 'react-native-webview';
import { useState, useEffect } from 'react';



export default function Call(props: {
  instanceUrl: string,
  token: string,
}){
  const { token, instanceUrl } = props;

  const [webViewRef, setWebViewRef] = useState(null);

  const injectToken = () => {
    if (!webViewRef) {
      return;
    }

    const jsCode = `window.localStorage.setItem('token', '${token}');`;

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
    setLastRefresh(Date.now());
  }

  useEffect(() => {
    const interval = setInterval(refresh, 30 * 60 * 1000);
    return () => clearInterval(interval);
  }, [lastRefresh]);

  return (
    <WebView
      style={styles.container}
      source={{ uri: `${instanceUrl}` }} // todo: add device path
      ref={setWebViewRef}
      onLoad={injectToken}
      injectedJavaScriptObject={{
        app: true,
        version: 1, // todo
        settings: {
          // todo
        }
    }}
    />
  )
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
  },
});
