import { StatusBar } from 'expo-status-bar';
import { StyleSheet, Text, View } from 'react-native';
import Enroll from "./views/Enroll";
import { useKeepAwake } from 'expo-keep-awake';

export default function App() {
  useKeepAwake();
  const enrolled = false;

  if (!enrolled) {
    return <Enroll />;
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
