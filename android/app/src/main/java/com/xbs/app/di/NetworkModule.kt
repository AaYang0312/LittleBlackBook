package com.xbs.app.di

import retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.xbs.app.BuildConfig
import com.xbs.app.data.api.AuthEventBus
import com.xbs.app.data.api.AuthInterceptor
import com.xbs.app.data.api.ErrorEnvelopeInterceptor
import com.xbs.app.data.api.FeedApi
import com.xbs.app.data.api.InteractionApi
import com.xbs.app.data.api.NoteApi
import com.xbs.app.data.api.UnauthorizedInterceptor
import com.xbs.app.data.api.UserApi
import com.xbs.app.data.local.DataStoreTokenStore
import com.xbs.app.data.local.TokenStore
import com.xbs.app.data.repository.FeedRepository
import com.xbs.app.data.repository.InteractionRepository
import com.xbs.app.data.repository.NoteRepository
import com.xbs.app.data.repository.UserRepository
import dagger.Binds
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
abstract class TokenStoreModule {
    @Binds
    @Singleton
    abstract fun bindTokenStore(impl: DataStoreTokenStore): TokenStore
}

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    @Provides
    @Singleton
    fun provideJson(): Json = Json { ignoreUnknownKeys = true }

    @Provides
    @Singleton
    fun provideOkHttp(
        authInterceptor: AuthInterceptor,
        unauthorizedInterceptor: UnauthorizedInterceptor,
    ): OkHttpClient = OkHttpClient.Builder()
        .addInterceptor(authInterceptor)
        .addInterceptor(unauthorizedInterceptor)
        .addInterceptor(ErrorEnvelopeInterceptor())
        .addInterceptor(HttpLoggingInterceptor().apply { level = HttpLoggingInterceptor.Level.BASIC })
        .build()

    @Provides
    @Singleton
    fun provideRetrofit(client: OkHttpClient, json: Json): Retrofit = Retrofit.Builder()
        .baseUrl(BuildConfig.API_BASE_URL)
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()

    @Provides @Singleton fun provideUserApi(r: Retrofit): UserApi = r.create(UserApi::class.java)
    @Provides @Singleton fun provideNoteApi(r: Retrofit): NoteApi = r.create(NoteApi::class.java)
    @Provides @Singleton fun provideFeedApi(r: Retrofit): FeedApi = r.create(FeedApi::class.java)
    @Provides @Singleton fun provideInteractionApi(r: Retrofit): InteractionApi = r.create(InteractionApi::class.java)
}

@Module
@InstallIn(SingletonComponent::class)
object RepositoryModule {

    @Provides @Singleton fun provideUserRepository(api: UserApi, tokenStore: TokenStore): UserRepository =
        UserRepository(api, tokenStore)

    @Provides @Singleton fun provideNoteRepository(api: NoteApi): NoteRepository = NoteRepository(api)

    @Provides @Singleton fun provideFeedRepository(api: FeedApi): FeedRepository = FeedRepository(api)

    @Provides @Singleton fun provideInteractionRepository(api: InteractionApi): InteractionRepository =
        InteractionRepository(api)
}
