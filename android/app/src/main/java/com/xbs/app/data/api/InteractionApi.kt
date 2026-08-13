package com.xbs.app.data.api

import com.xbs.app.data.api.dto.CommentDto
import com.xbs.app.data.api.dto.CommentReq
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

interface InteractionApi {
    @POST("users/{id}/follow") suspend fun follow(@Path("id") userId: Long): ApiResponse<Unit>
    @DELETE("users/{id}/follow") suspend fun unfollow(@Path("id") userId: Long): ApiResponse<Unit>
    @POST("notes/{id}/like") suspend fun like(@Path("id") noteId: Long): ApiResponse<Unit>
    @DELETE("notes/{id}/like") suspend fun unlike(@Path("id") noteId: Long): ApiResponse<Unit>
    @POST("notes/{id}/collect") suspend fun collect(@Path("id") noteId: Long): ApiResponse<Unit>
    @DELETE("notes/{id}/collect") suspend fun uncollect(@Path("id") noteId: Long): ApiResponse<Unit>

    @POST("notes/{id}/comments")
    suspend fun createComment(@Path("id") noteId: Long, @Body req: CommentReq): ApiResponse<CommentDto>

    @GET("notes/{id}/comments")
    suspend fun listComments(
        @Path("id") noteId: Long,
        @Query("cursor") cursor: Long? = null,
        @Query("size") size: Int = 20,
    ): ApiResponse<List<CommentDto>>
}
