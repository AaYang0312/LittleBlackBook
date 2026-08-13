package com.xbs.app.data.repository

import com.xbs.app.data.api.UserApi
import com.xbs.app.data.api.dataOrThrow
import com.xbs.app.data.api.dto.LoginReq
import com.xbs.app.data.api.dto.RegisterReq
import com.xbs.app.data.api.dto.UserDto
import com.xbs.app.data.local.TokenStore
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class UserRepository @Inject constructor(
    private val api: UserApi,
    private val tokenStore: TokenStore,
) {
    suspend fun login(username: String, password: String): Result<Unit> = runCatching {
        val token = api.login(LoginReq(username, password)).dataOrThrow().token
        tokenStore.save(token)
    }

    suspend fun register(username: String, password: String, nickname: String): Result<Unit> = runCatching {
        api.register(RegisterReq(username, password, nickname)).dataOrThrow()
    }

    suspend fun me(): Result<UserDto> = runCatching { api.me().dataOrThrow() }

    suspend fun logout() = tokenStore.clear()
}
