<!-- IP Whitelist Table Skeleton -->
<div class="animate-pulse">
	<!-- Settings Tabs Skeleton -->
	<div class="border-b border-gray-200 mb-6">
		<nav class="mb-2 flex space-x-8" aria-label="Tabs">
			<div class="h-6 bg-gray-300 rounded w-20"></div>
			<div class="h-6 bg-gray-300 rounded w-24"></div>
			<div class="h-6 bg-gray-300 rounded w-20"></div>
			<div class="h-6 bg-gray-300 rounded w-24"></div>
		</nav>
	</div>

	<!-- Status Section Skeleton -->
	<div class="mb-6 p-4 bg-gray-50 rounded-lg">
		<div class="flex items-center justify-between">
			<div class="flex items-center space-x-3">
				<div class="h-6 bg-gray-200 rounded w-32"></div>
				<div class="h-6 bg-gray-200 rounded w-12"></div>
			</div>
			<div class="h-8 bg-gray-200 rounded w-20"></div>
		</div>
		<div class="mt-2 h-4 bg-gray-200 rounded w-96"></div>
	</div>

	<!-- Table Skeleton -->
	<div class="mt-8 flow-root">
		<div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
			<div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
				<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
					<table class="min-w-full divide-y divide-gray-300">
						<thead class="bg-gray-50">
							<tr>
								<th class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6">
									<div class="h-4 bg-gray-200 rounded w-24"></div>
								</th>
								<th class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">
									<div class="h-4 bg-gray-200 rounded w-16"></div>
								</th>
								<th class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">
									<div class="h-4 bg-gray-200 rounded w-12"></div>
								</th>
								<th class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">
									<div class="h-4 bg-gray-200 rounded w-16"></div>
								</th>
								<th class="relative py-3.5 pl-3 pr-4 sm:pr-6">
									<div class="h-4 bg-gray-200 rounded w-8"></div>
								</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-200 bg-white">
							{#each Array(5)}
								<tr>
									<td
										class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6"
									>
										<div class="h-4 bg-gray-200 rounded w-32"></div>
									</td>
									<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
										<div class="h-4 bg-gray-200 rounded w-20"></div>
									</td>
									<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
										<div class="h-4 bg-gray-200 rounded w-24"></div>
									</td>
									<td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
										<div class="h-4 bg-gray-200 rounded w-20"></div>
									</td>
									<td
										class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
									>
										<div class="flex justify-end space-x-2">
											<div class="h-4 bg-gray-200 rounded w-16"></div>
											<div class="h-4 bg-gray-200 rounded w-8"></div>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	</div>
</div>
