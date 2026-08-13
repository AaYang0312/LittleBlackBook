package com.xbs.app.data.local

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

private val Context.dataStore by preferencesDataStore(name = "xbs_prefs")

interface TokenStore {
    val tokenFlow: Flow<String?>
    suspend fun token(): String?
    suspend fun save(token: String)
    suspend fun clear()
}

@Singleton
class DataStoreTokenStore @Inject constructor(
    @ApplicationContext context: Context,
) : TokenStore {
    private val store = context.dataStore
    private val key = stringPreferencesKey("jwt_token")

    override val tokenFlow: Flow<String?> = store.data.map { it[key] }
    override suspend fun token(): String? = tokenFlow.first()
    override suspend fun save(token: String) { store.edit { it[key] = token } }
    override suspend fun clear() { store.edit { it.remove(key) } }
}
