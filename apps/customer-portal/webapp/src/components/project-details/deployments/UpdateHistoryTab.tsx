// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import {
  Box,
  Button,
  IconButton,
  Skeleton,
  TextField,
  Typography,
  alpha,
} from "@wso2/oxygen-ui";
import { SquarePen, Trash2 } from "@wso2/oxygen-ui-icons-react";
import {
  useCallback,
  useMemo,
  useState,
  type ChangeEvent,
  type JSX,
} from "react";
import type { ProductUpdate } from "@models/responses";

export interface UpdateHistoryTabProps {
  updates: ProductUpdate[];
  isLoading?: boolean;
  onSaveUpdates: (updates: ProductUpdate[]) => Promise<void>;
}

interface UpdateFormData {
  updateLevel: string;
  date: string;
  details: string;
}

const INITIAL_FORM: UpdateFormData = {
  updateLevel: "",
  date: "",
  details: "",
};

/**
 * Displays update history timeline and allows adding/editing/deleting updates.
 *
 * @param {UpdateHistoryTabProps} props - updates array, loading state, and save callback.
 * @returns {JSX.Element} The update history tab component.
 */
export default function UpdateHistoryTab({
  updates,
  isLoading,
  onSaveUpdates,
}: UpdateHistoryTabProps): JSX.Element {
  const [form, setForm] = useState<UpdateFormData>(INITIAL_FORM);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const sortedUpdates = useMemo(() => {
    return [...updates].sort((a, b) => {
      const dateA = new Date(a.date).getTime();
      const dateB = new Date(b.date).getTime();
      return dateB - dateA;
    });
  }, [updates]);

  const currentUpdateLevel = useMemo(() => {
    if (sortedUpdates.length === 0) return null;
    const levels = sortedUpdates
      .map((u) => u.updateLevel)
      .filter((l): l is number => typeof l === "number");
    return levels.length > 0 ? Math.max(...levels) : null;
  }, [sortedUpdates]);

  const handleFormChange =
    (field: keyof UpdateFormData) => (e: ChangeEvent<HTMLInputElement>) => {
      setForm((prev) => ({ ...prev, [field]: e.target.value }));
    };

  const handleAddUpdate = useCallback(async () => {
    const updateLevel = parseInt(form.updateLevel, 10);
    if (!form.updateLevel || !form.date || isNaN(updateLevel)) {
      return;
    }

    const newUpdate: ProductUpdate = {
      updateLevel,
      date: form.date,
      details: form.details || undefined,
    };

    setIsSaving(true);
    try {
      await onSaveUpdates([...updates, newUpdate]);
      setForm(INITIAL_FORM);
    } finally {
      setIsSaving(false);
    }
  }, [form, updates, onSaveUpdates]);

  const handleEditClick = useCallback((index: number) => {
    setEditingIndex(index);
  }, []);

  const handleDeleteUpdate = useCallback(
    async (index: number) => {
      const newUpdates = updates.filter((_, i) => i !== index);
      setIsSaving(true);
      try {
        await onSaveUpdates(newUpdates);
      } finally {
        setIsSaving(false);
      }
    },
    [updates, onSaveUpdates],
  );

  const handleSaveEdit = useCallback(
    async (index: number, editedUpdate: ProductUpdate) => {
      const newUpdates = [...updates];
      newUpdates[index] = editedUpdate;
      setIsSaving(true);
      try {
        await onSaveUpdates(newUpdates);
        setEditingIndex(null);
      } finally {
        setIsSaving(false);
      }
    },
    [updates, onSaveUpdates],
  );

  const formatDate = (dateStr: string): string => {
    try {
      const date = new Date(dateStr);
      return date.toLocaleDateString("en-GB", {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
      });
    } catch {
      return dateStr;
    }
  };

  if (isLoading) {
    return <UpdateHistorySkeleton />;
  }

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
      {currentUpdateLevel !== null && (
        <Box
          sx={{
            bgcolor: (theme) => alpha(theme.palette.primary.main, 0.08),
            border: 1,
            borderColor: (theme) => alpha(theme.palette.primary.main, 0.3),
            p: 2,
          }}
        >
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
            }}
          >
            <Typography
              variant="body2"
              sx={{ fontWeight: 500, color: "text.primary" }}
            >
              Current Update Level:
            </Typography>
            <Typography
              variant="h6"
              sx={{ fontWeight: 600, color: "primary.main" }}
            >
              U{currentUpdateLevel}
            </Typography>
          </Box>
        </Box>
      )}

      <Box>
        <Typography
          variant="subtitle2"
          sx={{ mb: 2, fontWeight: 600, color: "text.primary" }}
        >
          Update History
        </Typography>
        {sortedUpdates.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>
            No update history available
          </Typography>
        ) : (
          <Box sx={{ position: "relative" }}>
            <Box
              sx={{
                position: "absolute",
                left: 13,
                top: 0,
                bottom: 0,
                width: "2px",
                bgcolor: "divider",
              }}
            />
            <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
              {sortedUpdates.map((update, index) => (
                <TimelineItem
                  key={index}
                  update={update}
                  isEditing={editingIndex === index}
                  onEdit={() => handleEditClick(index)}
                  onDelete={() => handleDeleteUpdate(index)}
                  onSave={(edited) => handleSaveEdit(index, edited)}
                  onCancelEdit={() => setEditingIndex(null)}
                  formatDate={formatDate}
                  isSaving={isSaving}
                />
              ))}
            </Box>
          </Box>
        )}
      </Box>

      <Box
        sx={{
          borderTop: 1,
          borderColor: "divider",
          pt: 3,
          display: "flex",
          flexDirection: "column",
          gap: 2,
        }}
      >
        <Typography
          variant="subtitle2"
          sx={{ fontWeight: 600, color: "text.primary" }}
        >
          Add New Update
        </Typography>
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            gap: 2,
          }}
        >
          <TextField
            id="new-update-level"
            label="Update Level *"
            placeholder="e.g., U25"
            value={form.updateLevel}
            onChange={handleFormChange("updateLevel")}
            fullWidth
            size="small"
            disabled={isSaving}
            type="number"
            inputProps={{ min: 0 }}
          />
          <TextField
            id="new-applied-on"
            label="Applied On *"
            type="date"
            value={form.date}
            onChange={handleFormChange("date")}
            fullWidth
            size="small"
            disabled={isSaving}
            slotProps={{ inputLabel: { shrink: true } }}
          />
        </Box>
        <TextField
          id="new-update-description"
          label="Description"
          placeholder="Brief description about the update..."
          value={form.details}
          onChange={handleFormChange("details")}
          fullWidth
          size="small"
          multiline
          rows={2}
          disabled={isSaving}
        />
        <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
          <Button
            variant="contained"
            color="primary"
            onClick={handleAddUpdate}
            disabled={!form.updateLevel || !form.date || isSaving}
            sx={{ minWidth: 120 }}
          >
            {isSaving ? "Adding..." : "Add Update"}
          </Button>
        </Box>
      </Box>
    </Box>
  );
}

interface TimelineItemProps {
  update: ProductUpdate;
  isEditing: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onSave: (edited: ProductUpdate) => void;
  onCancelEdit: () => void;
  formatDate: (dateStr: string) => string;
  isSaving: boolean;
}

/**
 * Single timeline item displaying an update entry.
 *
 * @param {TimelineItemProps} props - update data and event handlers.
 * @returns {JSX.Element} The timeline item component.
 */
function TimelineItem({
  update,
  isEditing,
  onEdit,
  onDelete,
  onSave,
  onCancelEdit,
  formatDate,
  isSaving,
}: TimelineItemProps): JSX.Element {
  const [editForm, setEditForm] = useState<ProductUpdate>(update);

  const handleEditChange =
    (field: keyof ProductUpdate) => (e: ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setEditForm((prev) => ({
        ...prev,
        [field]: field === "updateLevel" ? parseInt(value, 10) : value,
      }));
    };

  const handleSave = () => {
    if (editForm.updateLevel && editForm.date) {
      onSave(editForm);
    }
  };

  return (
    <Box sx={{ position: "relative", display: "flex", gap: 2 }}>
      <Box
        sx={{
          position: "relative",
          zIndex: 10,
          flexShrink: 0,
        }}
      >
        <Box
          sx={{
            width: 28,
            height: 28,
            bgcolor: "primary.main",
            border: 4,
            borderColor: "background.paper",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            borderRadius: "50%",
          }}
        >
          <Box
            sx={{
              width: 8,
              height: 8,
              bgcolor: "background.paper",
              borderRadius: "50%",
            }}
          />
        </Box>
      </Box>
      <Box
        sx={{
          flex: 1,
          bgcolor: (theme) =>
            theme.palette.mode === "dark" ? "grey.800" : "grey.700",
          color: "white",
          p: 2,
          position: "relative",
          ml: -1,
        }}
      >
        <Box
          sx={{
            position: "absolute",
            left: 0,
            top: 12,
            width: 0,
            height: 0,
            borderTop: "6px solid transparent",
            borderBottom: "6px solid transparent",
            borderRight: (theme) =>
              `8px solid ${theme.palette.mode === "dark" ? theme.palette.grey[800] : theme.palette.grey[700]}`,
            transform: "translateX(-100%)",
          }}
        />
        {isEditing ? (
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
            <Box
              sx={{
                display: "grid",
                gridTemplateColumns: "1fr 1fr",
                gap: 2,
              }}
            >
              <TextField
                label="Update Level"
                type="number"
                value={editForm.updateLevel}
                onChange={handleEditChange("updateLevel")}
                size="small"
                fullWidth
                disabled={isSaving}
                inputProps={{ min: 0 }}
                sx={{
                  "& .MuiInputBase-root": { bgcolor: "background.paper" },
                }}
              />
              <TextField
                label="Date"
                type="date"
                value={editForm.date}
                onChange={handleEditChange("date")}
                size="small"
                fullWidth
                disabled={isSaving}
                slotProps={{ inputLabel: { shrink: true } }}
                sx={{
                  "& .MuiInputBase-root": { bgcolor: "background.paper" },
                }}
              />
            </Box>
            <TextField
              label="Description"
              value={editForm.details || ""}
              onChange={handleEditChange("details")}
              size="small"
              fullWidth
              multiline
              rows={2}
              disabled={isSaving}
              sx={{
                "& .MuiInputBase-root": { bgcolor: "background.paper" },
              }}
            />
            <Box sx={{ display: "flex", justifyContent: "flex-end", gap: 1 }}>
              <Button
                variant="outlined"
                size="small"
                onClick={onCancelEdit}
                disabled={isSaving}
                sx={{
                  color: "grey.300",
                  borderColor: "grey.600",
                  "&:hover": { borderColor: "grey.400" },
                }}
              >
                Cancel
              </Button>
              <Button
                variant="contained"
                size="small"
                onClick={handleSave}
                disabled={isSaving || !editForm.updateLevel || !editForm.date}
                sx={{
                  bgcolor: "primary.main",
                  "&:hover": { bgcolor: "primary.dark" },
                }}
              >
                {isSaving ? "Saving..." : "Save"}
              </Button>
            </Box>
          </Box>
        ) : (
          <>
            <Box
              sx={{
                display: "flex",
                alignItems: "flex-start",
                justifyContent: "space-between",
                mb: 1,
              }}
            >
              <Box sx={{ flex: 1 }}>
                <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                  <Typography
                    variant="h6"
                    sx={{ fontWeight: 600, color: "white" }}
                  >
                    U{update.updateLevel}
                  </Typography>
                  <IconButton
                    size="small"
                    onClick={onEdit}
                    disabled={isSaving}
                    sx={{
                      color: "grey.300",
                      "&:hover": {
                        color: "primary.light",
                        bgcolor: (theme) => alpha(theme.palette.grey[600], 0.3),
                      },
                    }}
                  >
                    <SquarePen size={12} aria-hidden />
                  </IconButton>
                </Box>
                <Typography variant="body2" sx={{ color: "grey.300", mt: 0.5 }}>
                  Date: {formatDate(update.date)}
                </Typography>
              </Box>
              <IconButton
                size="small"
                onClick={onDelete}
                disabled={isSaving}
                sx={{
                  color: "grey.300",
                  "&:hover": {
                    color: "error.light",
                    bgcolor: (theme) => alpha(theme.palette.grey[600], 0.3),
                  },
                }}
              >
                <Trash2 size={16} aria-hidden />
              </IconButton>
            </Box>
            {update.details && (
              <Typography variant="body2" sx={{ color: "grey.200" }}>
                {update.details}
              </Typography>
            )}
          </>
        )}
      </Box>
    </Box>
  );
}

/**
 * Loading skeleton for update history tab.
 *
 * @returns {JSX.Element} The skeleton component.
 */
function UpdateHistorySkeleton(): JSX.Element {
  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
      <Skeleton variant="rectangular" width="100%" height={60} />
      <Box>
        <Skeleton variant="text" width={150} height={24} sx={{ mb: 2 }} />
        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {[1, 2].map((i) => (
            <Box key={i} sx={{ display: "flex", gap: 2 }}>
              <Skeleton
                variant="circular"
                width={28}
                height={28}
                sx={{ flexShrink: 0 }}
              />
              <Skeleton variant="rectangular" width="100%" height={100} />
            </Box>
          ))}
        </Box>
      </Box>
      <Box sx={{ borderTop: 1, borderColor: "divider", pt: 3 }}>
        <Skeleton variant="text" width={150} height={24} sx={{ mb: 2 }} />
        <Skeleton variant="rectangular" width="100%" height={120} />
      </Box>
    </Box>
  );
}
