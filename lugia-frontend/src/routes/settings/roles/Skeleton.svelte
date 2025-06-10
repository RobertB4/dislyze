<div class="animate-pulse">
	<!-- Settings Tabs Skeleton -->
	<div class="border-b border-gray-200 mb-6">
		<nav class="mb-2 flex space-x-8" aria-label="Tabs">
			<div class="h-6 bg-gray-300 rounded w-20"></div>
			<div class="h-6 bg-gray-300 rounded w-24"></div>
			<div class="h-6 bg-gray-300 rounded w-20"></div>
		</nav>
	</div>

	<!-- Table Skeleton -->
	<div class="mt-8 flow-root">
		<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
			<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
				<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
					<div class="min-w-full divide-y divide-gray-300">
						<!-- Table Header Skeleton -->
						<div class="bg-gray-50">
							<div class="grid grid-cols-5 gap-x-6 py-3.5 px-4 sm:px-6">
								<div class="h-4 bg-gray-300 rounded w-16"></div>
								<div class="h-4 bg-gray-300 rounded w-12"></div>
								<div class="h-4 bg-gray-300 rounded w-12"></div>
								<div class="h-4 bg-gray-300 rounded w-12"></div>
								<div class="h-4 bg-gray-300 rounded w-12"></div>
							</div>
						</div>
						<!-- Table Body Skeleton -->
						<div class="divide-y divide-gray-200 bg-white">
							{#each Array(4)}
								<div class="grid grid-cols-5 gap-x-6 py-4 px-4 sm:px-6 items-center">
									<!-- Role Name -->
									<div class="h-4 bg-gray-300 rounded w-20"></div>
									<!-- Description -->
									<div class="h-4 bg-gray-300 rounded w-32"></div>
									<!-- Permissions -->
									<div class="flex flex-wrap gap-1">
										<div class="h-6 bg-gray-200 rounded-md w-16"></div>
										<div class="h-6 bg-gray-200 rounded-md w-14"></div>
										<div class="h-6 bg-gray-200 rounded-md w-12"></div>
									</div>
									<!-- Type Badge -->
									<div class="h-6 bg-gray-200 rounded-md w-16"></div>
									<!-- Actions -->
									<div class="flex space-x-2 justify-end">
										<div class="h-4 bg-gray-200 rounded w-8"></div>
										<div class="h-4 bg-gray-200 rounded w-8"></div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
