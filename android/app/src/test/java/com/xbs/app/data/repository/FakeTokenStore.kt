package com.xbs.app.data.repository

import com.xbs.app.data.local.TokenStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow

class FakeTokenStore : TokenStore {
    private val state = MutableStateFlow<String?>(null)
    override val tokenFlow: Flow<String?> = state
    override suspend fun token(): String? = state.value
    override suspend fun save(token: String) { state.value = token }
    override suspend fun clear() { state.value = null }
}
