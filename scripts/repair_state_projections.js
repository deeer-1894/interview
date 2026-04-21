const now = new Date();

const completedRunsResult = db.runs.updateMany(
  { status: "completed", phase: { $nin: ["completed", null, ""] } },
  { $set: { phase: "completed" } },
);

const terminalRunsResult = db.runs.updateMany(
  {
    status: { $in: ["completed", "failed", "cancelled"] },
    completedat: null,
  },
  [
    {
      $set: {
        completedat: {
          $ifNull: ["$updatedat", now],
        },
      },
    },
  ],
);

let conversationProjectionUpdates = 0;
db.conversations
  .find({ latestrunid: { $exists: true, $ne: "" } }, { _id: 1, latestrunid: 1 })
  .forEach((conversation) => {
    const run = db.runs.findOne(
      { id: conversation.latestrunid },
      { status: 1, updatedat: 1 },
    );
    if (!run || !run.status) {
      return;
    }
    const result = db.conversations.updateOne(
      { _id: conversation._id },
      {
        $set: {
          latestrunstatus: run.status,
        },
      },
    );
    conversationProjectionUpdates += result.modifiedCount;
  });

printjson({
  completedRunsPhaseFixed: completedRunsResult.modifiedCount,
  terminalRunsCompletedAtFixed: terminalRunsResult.modifiedCount,
  conversationProjectionFixed: conversationProjectionUpdates,
});
