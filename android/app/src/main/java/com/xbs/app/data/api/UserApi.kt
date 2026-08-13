package com.xbs.app.data.api

import com.xbs.app.data.api.dto.LoginReq
import com.xbs.app.data.api.dto.LoginResp
import com.xbs.app.data.api.dto.RegisterReq
import com.xbs.app.data.api.dto.UserDto
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST

interface UserApi {
    @POST("users/register") suspend fun register(@Body req: RegisterReq): ApiResponse<UserDto>
    @POST("users/login") suspend fun login(@Body req: LoginReq): ApiResponse<LoginResp>
    @GET("users/me") suspend fun me(): ApiResponse<UserDto>
}
