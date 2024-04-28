import { StatusBar } from 'expo-status-bar';
import { StyleSheet, Text, View } from 'react-native';
import Enroll from "./views/Enroll";
import { useKeepAwake } from 'expo-keep-awake';
import { useState } from 'react';

export default function App() {
  useKeepAwake();
  let [enrolled, setEnrolled] = useState<boolean>(false);

  if (!enrolled) {
    return <Enroll
      onEnroll={(data) => {
        console.log('Enrolled!', data);
        setEnrolled(true);
      }}
    />;
  }

  return (
    <View style={styles.container}>
      <Text>Open up app.tsx to start working on your app!</Text>
      <StatusBar style="auto" />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
    alignItems: 'center',
    justifyContent: 'center',
  },
});
