package com.xbs.app.data.api

import com.xbs.app.data.api.dto.NoteDto
import retrofit2.http.GET
import retrofit2.http.Query

interface FeedApi {
    @GET("feed/following")
    suspend fun following(@Query("offset") offset: Int, @Query("size") size: Int = 20): ApiResponse<List<NoteDto>>
}
