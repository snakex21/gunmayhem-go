statdisplay.gotoAndStop(7);
dropgun.gotoAndStop(7);
testbtn.onRelease = function()
{
   if(statdisplay._currentframe != 7)
   {
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 10;
      _root.campaignmode = false;
      _root.mapnumber = 13;
      _root.gototest = true;
      _root.gototestnumber = dropgun._currentframe;
   }
};
