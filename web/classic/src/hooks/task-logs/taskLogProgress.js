/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export const mergeTaskLogProgress = (currentItems, refreshedItems) => {
  const pollingStateById = new Map(
    refreshedItems.map((item) => [
      item.id,
      {
        progress: item.progress,
        status: item.status,
        finish_time: item.finish_time,
      },
    ]),
  );

  let changed = false;
  const items = currentItems.map((item) => {
    const pollingState = pollingStateById.get(item.id);
    if (!pollingState) {
      return item;
    }

    if (
      pollingState.progress === item.progress &&
      pollingState.status === item.status &&
      pollingState.finish_time === item.finish_time
    ) {
      return item;
    }

    changed = true;
    return { ...item, ...pollingState };
  });

  return changed ? items : currentItems;
};
