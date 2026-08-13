package com.xbs.app.data.api

import com.xbs.app.data.local.TokenStore
import kotlinx.coroutines.runBlocking
import okhttp3.Interceptor
import okhttp3.Response
import javax.inject.Inject

class UnauthorizedInterceptor @Inject constructor(
    private val tokenStore: TokenStore,
    private val authEventBus: AuthEventBus,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val hadToken = !runBlocking { tokenStore.token() }.isNullOrBlank()
        val resp = chain.proceed(chain.request())
        if (resp.code == 401 && hadToken) {
            runBlocking { tokenStore.clear() }
            authEventBus.emitUnauthorized()
        }
        return resp
    }
}
