import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, FlatList, TextInput, SafeAreaView, StatusBar, Platform } from 'react-native';

const API_URL = "https://real-time-leaderboard-9n7b.onrender.com";

export default function App() {
  const [data, setData] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(false);

  // Initial Fetch (Leaderboard)
  useEffect(() => {
    fetchLeaderboard();
  }, []);

  // Search Effect (Debounced)
  useEffect(() => {
    const timer = setTimeout(() => {
      if (searchQuery.length > 0) {
        fetchSearch(searchQuery);
      } else {
        fetchLeaderboard();
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  const fetchLeaderboard = async () => {
    try {
      const res = await fetch(`${API_URL}/leaderboard`);
      const json = await res.json();
      if (Array.isArray(json)) {
        setData(json);
      }
    } catch (err) {
      console.error("Failed to fetch leaderboard", err);
    }
  };

  const fetchSearch = async (query) => {
    try {
      const res = await fetch(`${API_URL}/search?q=${query}`);
      const json = await res.json();
      if (Array.isArray(json)) {
        setData(json);
      }
    } catch (err) {
      console.error("Failed to search", err);
    }
  };

  const renderItem = ({ item }) => (
    <View style={styles.card}>
      <View style={styles.rankBadge}>
        <Text style={styles.rankText}>#{item.rank}</Text>
      </View>
      <View style={styles.userInfo}>
        <Text style={styles.username}>{item.username}</Text>
      </View>
      <View style={styles.ratingInfo}>
        <Text style={styles.rating}>{item.rating}</Text>
        <Text style={styles.ratingLabel}>PTS</Text>
      </View>
    </View>
  );

  return (
    <SafeAreaView style={styles.container}>
      <StatusBar barStyle="light-content" backgroundColor="#121212" />

      <View style={styles.header}>
        <Text style={styles.title}>🏆 Leaderboard</Text>
      </View>

      <View style={styles.searchContainer}>
        <TextInput
          style={styles.searchInput}
          placeholder="Search user..."
          placeholderTextColor="#666"
          value={searchQuery}
          onChangeText={setSearchQuery}
        />
      </View>

      <View style={styles.listHeader}>
        <Text style={styles.headerCol1}>Rank</Text>
        <Text style={styles.headerCol2}>User</Text>
        <Text style={styles.headerCol3}>Rating</Text>
      </View>

      <FlatList
        data={data}
        renderItem={renderItem}
        keyExtractor={(item) => item.username}
        contentContainerStyle={styles.listContent}
        showsVerticalScrollIndicator={false}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#121212',
    paddingTop: Platform.OS === 'android' ? 25 : 0,
  },
  header: {
    padding: 20,
    alignItems: 'center',
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#FFD700', // Gold
    letterSpacing: 1,
  },
  searchContainer: {
    paddingHorizontal: 16,
    marginBottom: 10,
  },
  searchInput: {
    backgroundColor: '#1E1E1E',
    color: '#FFF',
    padding: 12,
    borderRadius: 8,
    fontSize: 16,
    borderWidth: 1,
    borderColor: '#333',
  },
  listHeader: {
    flexDirection: 'row',
    paddingHorizontal: 20,
    paddingVertical: 10,
    borderBottomWidth: 1,
    borderBottomColor: '#333',
  },
  headerCol1: { color: '#888', width: 60, fontWeight: 'bold' },
  headerCol2: { color: '#888', flex: 1, fontWeight: 'bold' },
  headerCol3: { color: '#888', width: 70, textAlign: 'right', fontWeight: 'bold' },
  listContent: {
    paddingHorizontal: 16,
    paddingTop: 10,
  },
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#1E1E1E',
    marginBottom: 8,
    padding: 12,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#333',
  },
  rankBadge: {
    width: 60,
    justifyContent: 'center',
  },
  rankText: {
    color: '#BB86FC',
    fontWeight: 'bold',
    fontSize: 16,
  },
  userInfo: {
    flex: 1,
  },
  username: {
    color: '#FFF',
    fontSize: 16,
    fontWeight: '500',
  },
  ratingInfo: {
    alignItems: 'flex-end',
    width: 70,
  },
  rating: {
    color: '#03DAC6',
    fontWeight: 'bold',
    fontSize: 18,
  },
  ratingLabel: {
    color: '#555',
    fontSize: 10,
  },
});
